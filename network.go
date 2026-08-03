package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
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
	if args[0] != "test" {
		return errors.New("network поддерживает interfaces и test")
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
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return LatencyReport{}, errors.New("address должен иметь вид host:port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return LatencyReport{}, errors.New("некорректный TCP port")
	}
	if count < 3 || count > 100 || timeout < 100*time.Millisecond || timeout > 10*time.Second {
		return LatencyReport{}, errors.New("count должен быть 3–100, timeout — 100ms–10s")
	}
	if time.Duration(count)*timeout > 5*time.Minute {
		return LatencyReport{}, errors.New("максимальная длительность network test ограничена пятью минутами")
	}
	report := LatencyReport{Address: address, Attempts: count}
	for i := 0; i < count; i++ {
		started := time.Now()
		connection, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			report.Failed++
			continue
		}
		elapsed := float64(time.Since(started).Microseconds()) / 1000
		connection.Close()
		report.SamplesMS = append(report.SamplesMS, elapsed)
	}
	report.Succeeded = len(report.SamplesMS)
	if report.Succeeded == 0 {
		return report, errors.New("все TCP latency attempts завершились ошибкой")
	}
	ordered := append([]float64(nil), report.SamplesMS...)
	sort.Float64s(ordered)
	report.MinMS, report.MaxMS = ordered[0], ordered[len(ordered)-1]
	report.MedianMS = percentile(ordered, 0.5)
	report.P95MS = percentile(ordered, 0.95)
	if len(report.SamplesMS) > 1 {
		for i := 1; i < len(report.SamplesMS); i++ {
			report.JitterMS += math.Abs(report.SamplesMS[i] - report.SamplesMS[i-1])
		}
		report.JitterMS /= float64(len(report.SamplesMS) - 1)
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
