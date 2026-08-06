package startup

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type setRequest struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func Handle(_ context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "startup.list":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.Startup(), nil
	case "startup.set":
		request, err := protocol.DecodePayload[setRequest](payload)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"enabled": request.Enabled}, service.SetStartup(request.Name, request.Enabled)
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
