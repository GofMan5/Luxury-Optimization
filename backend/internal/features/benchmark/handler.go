package benchmark

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type compareRequest struct {
	Before optimizer.BenchmarkSet `json:"before"`
	After  optimizer.BenchmarkSet `json:"after"`
}

func Handle(_ context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	if method != "benchmark.compare" {
		return nil, protocol.ErrMethodNotFound
	}
	request, err := protocol.DecodePayload[compareRequest](payload)
	if err != nil {
		return nil, err
	}
	return service.CompareBenchmarks(request.Before, request.After)
}
