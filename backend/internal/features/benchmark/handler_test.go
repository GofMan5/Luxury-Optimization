package benchmark

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
)

func TestCompareRoutesThroughValidatedBenchmarkService(t *testing.T) {
	payload := json.RawMessage(`{"before":{"label":"stock","runs":[{"average_fps":100,"one_percent_low_fps":70,"p95_frame_ms":12},{"average_fps":101,"one_percent_low_fps":71,"p95_frame_ms":11.9},{"average_fps":99,"one_percent_low_fps":69,"p95_frame_ms":12.1}]},"after":{"label":"tuned","runs":[{"average_fps":110,"one_percent_low_fps":80,"p95_frame_ms":10},{"average_fps":111,"one_percent_low_fps":81,"p95_frame_ms":9.9},{"average_fps":109,"one_percent_low_fps":79,"p95_frame_ms":10.1}]}}`)
	result, err := Handle(context.Background(), optimizer.NewService(), "benchmark.compare", payload)
	if err != nil {
		t.Fatal(err)
	}
	comparison := result.(optimizer.BenchmarkComparison)
	if comparison.Verdict != "measurably_improved" || comparison.BeforeLabel != "stock" || comparison.AfterLabel != "tuned" {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}
}
