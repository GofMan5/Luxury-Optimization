package optimizer

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUDPDNSLatencyUsesValidatedResponses(t *testing.T) {
	connection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		buffer := make([]byte, 512)
		for range 5 {
			size, address, readErr := connection.ReadFrom(buffer)
			if readErr != nil {
				return
			}
			response := append([]byte(nil), buffer[:size]...)
			binary.BigEndian.PutUint16(response[2:4], 0x8180)
			_, _ = connection.WriteTo(response, address)
		}
	}()
	report, err := measureUDPDNSLatency(context.Background(), connection.LocalAddr().String(), 5, time.Second)
	if err != nil || report.Protocol != "dns_rfc1035" || report.Succeeded != 5 || report.Failed != 0 || report.P95MS < report.MedianMS {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	<-done
}

func TestBufferbloatValidationAndRatings(t *testing.T) {
	request := BufferbloatRequest{ProbeAddress: "1.1.1.1:443", DurationMS: 1500, Streams: 1}
	if err := validateBufferbloatRequest(request, 1500*time.Millisecond); err == nil {
		t.Fatal("unbounded short diagnostic accepted")
	}
	if err := validateDiagnosticURL("http://example.com/down"); err == nil {
		t.Fatal("plain HTTP endpoint accepted internally")
	}
	for increase, expected := range map[float64]string{5: "low", 6: "moderate", 21: "high", 51: "severe"} {
		if actual := classifyBufferbloat(increase); actual != expected {
			t.Fatalf("increase %.1f: %s", increase, actual)
		}
	}
}

func TestBufferbloatPhasesGenerateBoundedLocalLoad(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			_ = connection.Close()
		}
	}()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		chunk := make([]byte, 32<<10)
		if request.Method == http.MethodPost {
			for {
				_, readErr := request.Body.Read(chunk)
				if readErr != nil {
					break
				}
				time.Sleep(2 * time.Millisecond)
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		for range 200 {
			if _, err := writer.Write(chunk); err != nil {
				return
			}
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(2 * time.Millisecond)
		}
	}))
	defer server.Close()

	baseline, err := measureTCPLatency(listener.Addr().String(), 3, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"download", "upload"} {
		phase, err := runBufferbloatPhase(context.Background(), mode, server.URL, listener.Addr().String(), 700*time.Millisecond, 1, baseline)
		if err != nil || !phase.Supported || phase.Bytes < 64<<10 || phase.Latency.Succeeded < 3 {
			t.Fatalf("%s phase=%+v err=%v", mode, phase, err)
		}
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	started := time.Now()
	if _, err := runBufferbloatPhase(cancelCtx, "download", server.URL, listener.Addr().String(), 2*time.Second, 1, baseline); !errors.Is(err, context.Canceled) || time.Since(started) > time.Second {
		t.Fatalf("cancellation err=%v elapsed=%s", err, time.Since(started))
	}
}

func TestPatternReaderStopsAtDeclaredLength(t *testing.T) {
	reader := &patternReader{remaining: 100}
	data, err := io.ReadAll(reader)
	if err != nil || len(data) != 100 || reader.remaining != 0 {
		t.Fatalf("bytes=%d remaining=%d err=%v", len(data), reader.remaining, err)
	}
}
