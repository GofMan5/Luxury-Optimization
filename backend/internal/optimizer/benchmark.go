package optimizer

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"sort"
)

type BenchmarkRun struct {
	AverageFPS float64 `json:"average_fps"`
	Low1FPS    float64 `json:"one_percent_low_fps"`
	P95FrameMS float64 `json:"p95_frame_ms"`
}

type BenchmarkSet struct {
	Label string         `json:"label"`
	Runs  []BenchmarkRun `json:"runs"`
}

type MetricComparison struct {
	BeforeMedian float64 `json:"before_median"`
	AfterMedian  float64 `json:"after_median"`
	DeltaPercent float64 `json:"delta_percent"`
	NoisePercent float64 `json:"noise_percent"`
	Meaningful   bool    `json:"meaningful"`
}

type BenchmarkComparison struct {
	BeforeLabel string           `json:"before_label"`
	AfterLabel  string           `json:"after_label"`
	AverageFPS  MetricComparison `json:"average_fps"`
	Low1FPS     MetricComparison `json:"one_percent_low_fps"`
	P95FrameMS  MetricComparison `json:"p95_frame_ms"`
	Verdict     string           `json:"verdict"`
}

const maxBenchmarkMetric = 100_000

func benchmarkCommand(args []string) error {
	if len(args) == 0 || args[0] == "template" {
		if len(args) > 1 {
			return errors.New("лишние аргументы benchmark template")
		}
		example := BenchmarkSet{Label: "before", Runs: []BenchmarkRun{{AverageFPS: 144, Low1FPS: 100, P95FrameMS: 8.5}, {AverageFPS: 143, Low1FPS: 99, P95FrameMS: 8.7}, {AverageFPS: 145, Low1FPS: 101, P95FrameMS: 8.4}}}
		data, _ := json.MarshalIndent(example, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	if args[0] != "compare" {
		return errors.New("benchmark поддерживает template и compare")
	}
	set := flag.NewFlagSet("benchmark compare", flag.ContinueOnError)
	beforePath := set.String("before", "", "JSON с исходными прогонами")
	afterPath := set.String("after", "", "JSON с прогонами после изменения")
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 || *beforePath == "" || *afterPath == "" {
		return errors.New("укажите --before и --after")
	}
	before, err := readBenchmarkSet(*beforePath)
	if err != nil {
		return fmt.Errorf("before: %w", err)
	}
	after, err := readBenchmarkSet(*afterPath)
	if err != nil {
		return fmt.Errorf("after: %w", err)
	}
	comparison := compareBenchmarks(before, after)
	if *jsonOnly {
		data, err := json.MarshalIndent(comparison, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Average FPS: %.2f → %.2f (%+.2f%%, noise %.2f%%)\n", comparison.AverageFPS.BeforeMedian, comparison.AverageFPS.AfterMedian, comparison.AverageFPS.DeltaPercent, comparison.AverageFPS.NoisePercent)
	fmt.Printf("1%% low FPS: %.2f → %.2f (%+.2f%%, noise %.2f%%)\n", comparison.Low1FPS.BeforeMedian, comparison.Low1FPS.AfterMedian, comparison.Low1FPS.DeltaPercent, comparison.Low1FPS.NoisePercent)
	fmt.Printf("p95 frametime: %.2f → %.2f ms (%+.2f%% better, noise %.2f%%)\n", comparison.P95FrameMS.BeforeMedian, comparison.P95FrameMS.AfterMedian, comparison.P95FrameMS.DeltaPercent, comparison.P95FrameMS.NoisePercent)
	fmt.Println("Verdict:", comparison.Verdict)
	return nil
}

func readBenchmarkSet(path string) (BenchmarkSet, error) {
	data, err := readSmallFile(path, 1<<20)
	if err != nil {
		return BenchmarkSet{}, err
	}
	var set BenchmarkSet
	if err := json.Unmarshal(data, &set); err != nil {
		return set, err
	}
	return set, validateBenchmarkSet(set)
}

func validateBenchmarkSet(set BenchmarkSet) error {
	if len(set.Runs) < 3 || len(set.Runs) > 100 {
		return errors.New("нужно от 3 до 100 одинаковых прогонов")
	}
	for index, run := range set.Runs {
		for _, value := range []float64{run.AverageFPS, run.Low1FPS, run.P95FrameMS} {
			if value <= 0 || value > maxBenchmarkMetric || math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("прогон %d содержит некорректную метрику", index+1)
			}
		}
	}
	return nil
}

func compareBenchmarks(before, after BenchmarkSet) BenchmarkComparison {
	metric := func(beforeValues, afterValues []float64, lowerIsBetter bool) MetricComparison {
		beforeMedian, afterMedian := median(beforeValues), median(afterValues)
		delta := (afterMedian - beforeMedian) / beforeMedian * 100
		if lowerIsBetter {
			delta = -delta
		}
		noise := math.Max(2, math.Max(madPercent(beforeValues), madPercent(afterValues))*2)
		return MetricComparison{BeforeMedian: beforeMedian, AfterMedian: afterMedian, DeltaPercent: delta, NoisePercent: noise, Meaningful: math.Abs(delta) > noise}
	}
	averageBefore, lowBefore, frameBefore := benchmarkColumns(before.Runs)
	averageAfter, lowAfter, frameAfter := benchmarkColumns(after.Runs)
	comparison := BenchmarkComparison{
		BeforeLabel: before.Label,
		AfterLabel:  after.Label,
		AverageFPS:  metric(averageBefore, averageAfter, false),
		Low1FPS:     metric(lowBefore, lowAfter, false),
		P95FrameMS:  metric(frameBefore, frameAfter, true),
		Verdict:     "within_run_to_run_variance",
	}
	meaningful := []MetricComparison{comparison.AverageFPS, comparison.Low1FPS, comparison.P95FrameMS}
	positive, negative := 0, 0
	for _, item := range meaningful {
		if item.Meaningful && item.DeltaPercent > 0 {
			positive++
		}
		if item.Meaningful && item.DeltaPercent < 0 {
			negative++
		}
	}
	if positive > 0 && negative == 0 {
		comparison.Verdict = "measurably_improved"
	} else if negative > 0 && positive == 0 {
		comparison.Verdict = "measurably_regressed"
	} else if positive > 0 || negative > 0 {
		comparison.Verdict = "mixed_result"
	}
	return comparison
}

func benchmarkColumns(runs []BenchmarkRun) ([]float64, []float64, []float64) {
	average, low, frame := make([]float64, len(runs)), make([]float64, len(runs)), make([]float64, len(runs))
	for i, run := range runs {
		average[i], low[i], frame[i] = run.AverageFPS, run.Low1FPS, run.P95FrameMS
	}
	return average, low, frame
}

func median(values []float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	if len(ordered)%2 == 0 {
		return (ordered[middle-1] + ordered[middle]) / 2
	}
	return ordered[middle]
}

func madPercent(values []float64) float64 {
	center := median(values)
	deviations := make([]float64, len(values))
	for i, value := range values {
		deviations[i] = math.Abs(value - center)
	}
	return median(deviations) / center * 100
}
