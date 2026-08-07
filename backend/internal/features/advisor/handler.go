package advisor

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type backgroundRequest struct {
	SampleMS int `json:"sample_ms,omitempty"`
}

func Handle(ctx context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	if method != "advisor.background" {
		return nil, protocol.ErrMethodNotFound
	}
	request, err := protocol.DecodePayload[backgroundRequest](payload)
	if err != nil {
		return nil, err
	}
	return service.BackgroundAdvisor(ctx, request.SampleMS)
}
