package optimizer

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bufferbloatPhaseBytes  = int64(128 << 20)
	bufferbloatRequestSize = int64(8 << 20)
	bufferbloatDownloadURL = "https://speed.cloudflare.com/__down?bytes=67108864"
	bufferbloatUploadURL   = "https://speed.cloudflare.com/__up"
)

type UDPLatencyReport struct {
	Protocol string `json:"protocol"`
	Question string `json:"question"`
	LatencyReport
}

type BufferbloatRequest struct {
	ProbeAddress string `json:"probe_address"`
	DurationMS   int    `json:"duration_ms"`
	Streams      int    `json:"streams"`
}

type BufferbloatPhase struct {
	Supported        bool          `json:"supported"`
	Reason           string        `json:"reason,omitempty"`
	Bytes            int64         `json:"bytes"`
	ThroughputMbps   float64       `json:"throughput_mbps"`
	Latency          LatencyReport `json:"latency"`
	P95IncreaseMS    float64       `json:"p95_increase_ms"`
	MedianIncreaseMS float64       `json:"median_increase_ms"`
	Rating           string        `json:"rating,omitempty"`
}

type BufferbloatReport struct {
	ProbeAddress string           `json:"probe_address"`
	DurationMS   int              `json:"duration_ms"`
	Streams      int              `json:"streams"`
	Baseline     LatencyReport    `json:"baseline"`
	Download     BufferbloatPhase `json:"download"`
	Upload       BufferbloatPhase `json:"upload"`
	Warnings     []string         `json:"warnings,omitempty"`
}

func measureUDPDNSLatency(ctx context.Context, address string, count int, timeout time.Duration) (UDPLatencyReport, error) {
	if err := validateHostPort(address, true); err != nil {
		return UDPLatencyReport{}, err
	}
	if count < 3 || count > 50 || timeout < 100*time.Millisecond || timeout > 5*time.Second {
		return UDPLatencyReport{}, errors.New("count must be 3-50 and timeout must be 100ms-5s")
	}
	if time.Duration(count)*timeout > 2*time.Minute {
		return UDPLatencyReport{}, errors.New("maximum UDP test duration is two minutes")
	}

	samples := make([]float64, 0, count)
	failed := 0
	for range count {
		if err := ctx.Err(); err != nil {
			return UDPLatencyReport{}, err
		}
		transaction := make([]byte, 2)
		if _, err := rand.Read(transaction); err != nil {
			return UDPLatencyReport{}, fmt.Errorf("UDP transaction ID: %w", err)
		}
		query := dnsQuery(binary.BigEndian.Uint16(transaction))
		started := time.Now()
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		connection, err := (&net.Dialer{}).DialContext(attemptCtx, "udp", address)
		if err == nil {
			deadline, _ := attemptCtx.Deadline()
			err = connection.SetDeadline(deadline)
			if err == nil {
				_, err = connection.Write(query)
			}
			response := make([]byte, 1232)
			var size int
			if err == nil {
				size, err = connection.Read(response)
			}
			_ = connection.Close()
			if err == nil {
				err = validateDNSResponse(response[:size], binary.BigEndian.Uint16(transaction))
			}
		}
		cancel()
		if ctx.Err() != nil {
			return UDPLatencyReport{}, ctx.Err()
		}
		if err != nil {
			failed++
			continue
		}
		samples = append(samples, milliseconds(time.Since(started)))
	}
	report := summarizeLatency(address, count, samples, failed)
	if report.Succeeded == 0 {
		return UDPLatencyReport{Protocol: "dns_rfc1035", Question: "example.com A", LatencyReport: report}, errors.New("all UDP DNS attempts failed")
	}
	return UDPLatencyReport{Protocol: "dns_rfc1035", Question: "example.com A", LatencyReport: report}, nil
}

func dnsQuery(transaction uint16) []byte {
	query := make([]byte, 0, 29)
	query = binary.BigEndian.AppendUint16(query, transaction)
	query = binary.BigEndian.AppendUint16(query, 0x0100) // recursion desired
	query = binary.BigEndian.AppendUint16(query, 1)
	query = append(query, make([]byte, 6)...)
	for _, label := range []string{"example", "com"} {
		query = append(query, byte(len(label)))
		query = append(query, label...)
	}
	query = append(query, 0)
	query = binary.BigEndian.AppendUint16(query, 1) // A
	return binary.BigEndian.AppendUint16(query, 1)  // IN
}

func validateDNSResponse(response []byte, transaction uint16) error {
	if len(response) < 12 || binary.BigEndian.Uint16(response) != transaction {
		return errors.New("invalid UDP DNS response")
	}
	flags := binary.BigEndian.Uint16(response[2:4])
	if flags&0x8000 == 0 || flags&0x000f != 0 || binary.BigEndian.Uint16(response[4:6]) == 0 {
		return errors.New("UDP DNS server returned an invalid response")
	}
	return nil
}

func measureBufferbloat(ctx context.Context, request BufferbloatRequest) (BufferbloatReport, error) {
	duration := time.Duration(request.DurationMS) * time.Millisecond
	if err := validateBufferbloatRequest(request, duration); err != nil {
		return BufferbloatReport{}, err
	}
	baseline, err := measureTCPLatencyContext(ctx, request.ProbeAddress, 5, 1500*time.Millisecond)
	if err != nil {
		return BufferbloatReport{}, fmt.Errorf("baseline latency: %w", err)
	}
	if baseline.Succeeded < 3 {
		return BufferbloatReport{}, errors.New("baseline latency requires at least three successful probes")
	}
	report := BufferbloatReport{ProbeAddress: request.ProbeAddress, DurationMS: request.DurationMS, Streams: request.Streams, Baseline: baseline}
	report.Download, err = runBufferbloatPhase(ctx, "download", bufferbloatDownloadURL, request.ProbeAddress, duration, request.Streams, baseline)
	if err != nil {
		return BufferbloatReport{}, err
	}
	report.Upload, err = runBufferbloatPhase(ctx, "upload", bufferbloatUploadURL, request.ProbeAddress, duration, request.Streams, baseline)
	if err != nil {
		return BufferbloatReport{}, err
	}
	if !report.Download.Supported {
		report.Warnings = append(report.Warnings, "download: "+report.Download.Reason)
	}
	if !report.Upload.Supported {
		report.Warnings = append(report.Warnings, "upload: "+report.Upload.Reason)
	}
	return report, nil
}

func validateBufferbloatRequest(request BufferbloatRequest, duration time.Duration) error {
	if err := validateHostPort(request.ProbeAddress, false); err != nil {
		return fmt.Errorf("probe_address: %w", err)
	}
	if duration < 2*time.Second || duration > 15*time.Second || request.Streams < 1 || request.Streams > 4 {
		return errors.New("duration_ms must be 2000-15000 and streams must be 1-4")
	}
	return nil
}

func validateDiagnosticURL(raw string) error {
	if len(raw) == 0 || len(raw) > 2048 {
		return errors.New("HTTPS URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "speed.cloudflare.com" || (parsed.Port() != "" && parsed.Port() != "443") || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("URL must use the approved HTTPS diagnostic host without credentials or a fragment")
	}
	return nil
}

func runBufferbloatPhase(parent context.Context, mode, endpoint, probeAddress string, duration time.Duration, streams int, baseline LatencyReport) (BufferbloatPhase, error) {
	ctx, cancel := context.WithTimeout(parent, duration)
	defer cancel()
	client := diagnosticHTTPClient(streams)
	defer client.CloseIdleConnections()

	var bytesTransferred atomic.Int64
	var firstErr error
	var errorMu sync.Mutex
	workerBudget := bufferbloatPhaseBytes / int64(streams)
	var workers sync.WaitGroup
	workers.Add(streams)
	for range streams {
		go func() {
			defer workers.Done()
			var err error
			if mode == "download" {
				err = loadDownload(ctx, client, endpoint, workerBudget, &bytesTransferred)
			} else {
				err = loadUpload(ctx, client, endpoint, workerBudget, &bytesTransferred)
			}
			if err != nil && ctx.Err() == nil {
				errorMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errorMu.Unlock()
			}
		}()
	}
	workersDone := make(chan struct{})
	go func() {
		workers.Wait()
		close(workersDone)
	}()

	started := time.Now()
	warmup := time.NewTimer(150 * time.Millisecond)
	select {
	case <-parent.Done():
		warmup.Stop()
		cancel()
		<-workersDone
		return BufferbloatPhase{}, parent.Err()
	case <-workersDone:
		warmup.Stop()
	case <-warmup.C:
	}

	samples := make([]float64, 0, 64)
	attempts, failed := 0, 0
	probing := true
	for probing && len(samples)+failed < 64 {
		select {
		case <-parent.Done():
			cancel()
			<-workersDone
			return BufferbloatPhase{}, parent.Err()
		case <-workersDone:
			probing = false
			continue
		case <-ctx.Done():
			probing = false
			continue
		default:
		}
		attempts++
		elapsed, err := probeTCP(ctx, probeAddress, 750*time.Millisecond)
		if err != nil {
			failed++
		} else {
			samples = append(samples, elapsed)
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			stopTimer(timer)
			probing = false
		case <-workersDone:
			stopTimer(timer)
			probing = false
		}
	}
	cancel()
	<-workersDone
	if err := parent.Err(); err != nil {
		return BufferbloatPhase{}, err
	}

	transferred := bytesTransferred.Load()
	elapsedSeconds := time.Since(started).Seconds()
	phase := BufferbloatPhase{Bytes: transferred, Latency: summarizeLatency(probeAddress, attempts, samples, failed)}
	if transferred < 64<<10 {
		if firstErr != nil {
			phase.Reason = displayText(firstErr.Error())
		} else {
			phase.Reason = "load endpoint transferred less than 64 KiB"
		}
		return phase, nil
	}
	if phase.Latency.Succeeded < 3 {
		phase.Reason = "fewer than three loaded latency probes succeeded"
		return phase, nil
	}
	phase.Supported = true
	phase.ThroughputMbps = float64(transferred) * 8 / elapsedSeconds / 1_000_000
	phase.P95IncreaseMS = math.Max(0, phase.Latency.P95MS-baseline.P95MS)
	phase.MedianIncreaseMS = math.Max(0, phase.Latency.MedianMS-baseline.MedianMS)
	phase.Rating = classifyBufferbloat(phase.P95IncreaseMS)
	return phase, nil
}

func diagnosticHTTPClient(streams int) *http.Client {
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          streams,
		MaxIdleConnsPerHost:   streams,
		MaxConnsPerHost:       streams,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		DisableCompression:    true,
	}
	return &http.Client{Transport: transport, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		return validateDiagnosticURL(request.URL.String())
	}}
}

func loadDownload(ctx context.Context, client *http.Client, endpoint string, limit int64, total *atomic.Int64) error {
	remaining := limit
	buffer := make([]byte, 64<<10)
	for remaining > 0 && ctx.Err() == nil {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		request.Header.Set("Accept-Encoding", "identity")
		request.Header.Set("User-Agent", "Luxury-Optimization-network-diagnostic")
		response, err := client.Do(request)
		if err != nil {
			return err
		}
		if response.StatusCode < 200 || response.StatusCode > 299 {
			if closeErr := response.Body.Close(); closeErr != nil {
				return fmt.Errorf("HTTP status %d; close response: %w", response.StatusCode, closeErr)
			}
			return fmt.Errorf("HTTP status %d", response.StatusCode)
		}
		writer := &countingLimitWriter{remaining: remaining, total: total}
		_, copyErr := io.CopyBuffer(writer, response.Body, buffer)
		closeErr := response.Body.Close()
		remaining = writer.remaining
		if copyErr != nil && !errors.Is(copyErr, errTransferLimit) {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func loadUpload(ctx context.Context, client *http.Client, endpoint string, limit int64, total *atomic.Int64) error {
	remaining := limit
	for remaining > 0 && ctx.Err() == nil {
		size := min(remaining, bufferbloatRequestSize)
		body := &patternReader{remaining: size, total: total}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, io.NopCloser(body))
		if err != nil {
			return err
		}
		request.ContentLength = size
		request.Header.Set("Content-Type", "application/octet-stream")
		request.Header.Set("User-Agent", "Luxury-Optimization-network-diagnostic")
		response, err := client.Do(request)
		remaining -= size - body.remaining
		if err != nil {
			return err
		}
		_, _ = io.CopyN(io.Discard, response.Body, 64<<10)
		if closeErr := response.Body.Close(); closeErr != nil {
			return closeErr
		}
		if response.StatusCode < 200 || response.StatusCode > 299 {
			return fmt.Errorf("HTTP status %d", response.StatusCode)
		}
	}
	return nil
}

var errTransferLimit = errors.New("transfer byte limit reached")

type countingLimitWriter struct {
	remaining int64
	total     *atomic.Int64
}

func (writer *countingLimitWriter) Write(value []byte) (int, error) {
	if writer.remaining == 0 {
		return 0, errTransferLimit
	}
	size := min(int64(len(value)), writer.remaining)
	writer.remaining -= size
	writer.total.Add(size)
	if size < int64(len(value)) || writer.remaining == 0 {
		return int(size), errTransferLimit
	}
	return int(size), nil
}

type patternReader struct {
	remaining int64
	total     *atomic.Int64
}

func (reader *patternReader) Read(value []byte) (int, error) {
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	size := min(int64(len(value)), reader.remaining)
	for index := range value[:size] {
		value[index] = byte(index*31 + 17)
	}
	reader.remaining -= size
	if reader.total != nil {
		reader.total.Add(size)
	}
	return int(size), nil
}

func probeTCP(ctx context.Context, address string, timeout time.Duration) (float64, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	connection, err := (&net.Dialer{}).DialContext(attemptCtx, "tcp", address)
	if err != nil {
		return 0, err
	}
	_ = connection.Close()
	return milliseconds(time.Since(started)), nil
}

func classifyBufferbloat(increaseMS float64) string {
	switch {
	case increaseMS <= 5:
		return "low"
	case increaseMS <= 20:
		return "moderate"
	case increaseMS <= 50:
		return "high"
	default:
		return "severe"
	}
}

func validateHostPort(address string, requireIP bool) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return errors.New("address must have the form host:port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("invalid port")
	}
	if requireIP && net.ParseIP(strings.Trim(host, "[]")) == nil {
		return errors.New("UDP DNS server must be an IP address")
	}
	return nil
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func summarizeLatency(address string, attempts int, samples []float64, failed int) LatencyReport {
	report := LatencyReport{Address: address, Attempts: attempts, Succeeded: len(samples), Failed: failed, SamplesMS: samples}
	if len(samples) == 0 {
		return report
	}
	ordered := append([]float64(nil), samples...)
	sort.Float64s(ordered)
	report.MinMS, report.MaxMS = ordered[0], ordered[len(ordered)-1]
	report.MedianMS = percentile(ordered, 0.5)
	report.P95MS = percentile(ordered, 0.95)
	if len(samples) > 1 {
		for index := 1; index < len(samples); index++ {
			report.JitterMS += math.Abs(samples[index] - samples[index-1])
		}
		report.JitterMS /= float64(len(samples) - 1)
	}
	return report
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}
