package services

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type listRequest struct {
	State string `json:"state,omitempty"`
	Match string `json:"match,omitempty"`
}

type setRequest struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func Handle(_ context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "services.list":
		request, err := protocol.DecodePayload[listRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.Services(request.State, request.Match)
	case "services.set":
		request, err := protocol.DecodePayload[setRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.SetService(request.Name, request.Enabled)
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
