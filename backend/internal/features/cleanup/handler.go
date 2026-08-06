package cleanup

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type runRequest struct {
	Days int `json:"days"`
}

func Handle(_ context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	if method != "cleanup.run" {
		return nil, protocol.ErrMethodNotFound
	}
	request, err := protocol.DecodePayload[runRequest](payload)
	if err != nil {
		return nil, err
	}
	return service.Clean(request.Days)
}
