package optimizer

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"net"
	"sort"
	"strings"
	"time"
)

type NetworkInterface struct {
	Index     int      `json:"index"`
	Name      string   `json:"name"`
	MTU       int      `json:"mtu"`
	Flags     string   `json:"flags"`
	Addresses []string `json:"addresses"`
}

type LatencyReport struct {
	Address   string    `json:"address"`
	Attempts  int       `json:"attempts"`
	Succeeded int       `json:"succeeded"`
	Failed    int       `json:"failed"`
	MinMS     float64   `json:"min_ms"`
	MedianMS  float64   `json:"median_ms"`
	P95MS     float64   `json:"p95_ms"`
	MaxMS     float64   `json:"max_ms"`
	JitterMS  float64   `json:"jitter_ms"`
	SamplesMS []float64 `json:"samples_ms"`
}

func networkCommand(args []string) error {
	if len(args) == 0 || args[0] == "interfaces" {
		if len(args) > 0 {
			args = args[1:]
		}
		set := flag.NewFlagSet("network interfaces", flag.ContinueOnError)
		jsonOnly := set.Bool("json", false, "вывести JSON")
		if err := set.Parse(args); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы network interfaces")
		}
		interfaces, err := listNetworkInterfaces()
		if err != nil {
			return err
		}
		if *jsonOnly {
			data, err := json.MarshalIndent(interfaces, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		for _, item := range interfaces {
			fmt.Printf("%d  %s  MTU=%d  %s\n  %s\n", item.Index, displayText(item.Name), item.MTU, displayText(item.Flags), displayText(strings.Join(item.Addresses, ", ")))
		}
		return nil
	}
	if args[0] == "udp" {
		set := flag.NewFlagSet("network udp", flag.ContinueOnError)
		address := set.String("address", "1.1.1.1:53", "DNS server IP:port")
		count := set.Int("count", 10, "число DNS запросов, 3–50")
		timeout := set.Duration("timeout", 2*time.Second, "таймаут одного DNS запроса")
		jsonOnly := set.Bool("json", false, "вывести JSON")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы network udp")
		}
		report, err := measureUDPDNSLatency(context.Background(), *address, *count, *timeout)
		if err != nil {
			return err
		}
		if *jsonOnly {
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("UDP DNS %s: median %.2f ms, p95 %.2f ms, jitter %.2f ms, failures %d/%d\n", displayText(report.Address), report.MedianMS, report.P95MS, report.JitterMS, report.Failed, report.Attempts)
		return nil
	}
	if args[0] == "bufferbloat" {
		set := flag.NewFlagSet("network bufferbloat", flag.ContinueOnError)
		probeAddress := set.String("probe", "1.1.1.1:443", "TCP latency probe host:port")
		duration := set.Duration("duration", 3*time.Second, "duration per loaded phase, 2s–15s")
		streams := set.Int("streams", 1, "parallel streams, 1–4")
		jsonOnly := set.Bool("json", false, "вывести JSON")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы network bufferbloat")
		}
		report, err := measureBufferbloat(context.Background(), BufferbloatRequest{ProbeAddress: *probeAddress, DurationMS: int(duration.Milliseconds()), Streams: *streams})
		if err != nil {
			return err
		}
		if *jsonOnly {
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("Bufferbloat %s: download %s (+%.2f ms p95), upload %s (+%.2f ms p95)\n", displayText(report.ProbeAddress), report.Download.Rating, report.Download.P95IncreaseMS, report.Upload.Rating, report.Upload.P95IncreaseMS)
		return nil
	}
	if args[0] != "test" {
		return errors.New("network поддерживает interfaces, test, udp и bufferbloat")
	}
	set := flag.NewFlagSet("network test", flag.ContinueOnError)
	address := set.String("address", "1.1.1.1:443", "host:port")
	count := set.Int("count", 10, "число TCP соединений, 3–100")
	timeout := set.Duration("timeout", 2*time.Second, "таймаут одного соединения")
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы network test")
	}
	report, err := measureTCPLatency(*address, *count, *timeout)
	if err != nil {
		return err
	}
	if *jsonOnly {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("TCP %s: median %.2f ms, p95 %.2f ms, jitter %.2f ms, failures %d/%d\n", displayText(report.Address), report.MedianMS, report.P95MS, report.JitterMS, report.Failed, report.Attempts)
	return nil
}

func listNetworkInterfaces() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	result := make([]NetworkInterface, 0, len(interfaces))
	for _, item := range interfaces {
		addresses, _ := item.Addrs()
		entry := NetworkInterface{Index: item.Index, Name: item.Name, MTU: item.MTU, Flags: item.Flags.String()}
		for _, address := range addresses {
			entry.Addresses = append(entry.Addresses, address.String())
		}
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result, nil
}

func measureTCPLatency(address string, count int, timeout time.Duration) (LatencyReport, error) {
	return measureTCPLatencyContext(context.Background(), address, count, timeout)
}

func measureTCPLatencyContext(ctx context.Context, address string, count int, timeout time.Duration) (LatencyReport, error) {
	if err := validateHostPort(address, false); err != nil {
		return LatencyReport{}, err
	}
	if count < 3 || count > 100 || timeout < 100*time.Millisecond || timeout > 10*time.Second {
		return LatencyReport{}, errors.New("count должен быть 3–100, timeout — 100ms–10s")
	}
	if time.Duration(count)*timeout > 5*time.Minute {
		return LatencyReport{}, errors.New("максимальная длительность network test ограничена пятью минутами")
	}
	samples := make([]float64, 0, count)
	failed := 0
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			return LatencyReport{}, err
		}
		elapsed, err := probeTCP(ctx, address, timeout)
		if err != nil {
			failed++
			continue
		}
		samples = append(samples, elapsed)
	}
	report := summarizeLatency(address, count, samples, failed)
	if report.Succeeded == 0 {
		return report, errors.New("все TCP latency attempts завершились ошибкой")
	}
	return report, nil
}

func percentile(sorted []float64, fraction float64) float64 {
	index := int(math.Ceil(fraction*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
