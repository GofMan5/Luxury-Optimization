package updates

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

func Handle(ctx context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "updates.status", "updates.check", "updates.install":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		switch method {
		case "updates.status":
			return service.UpdateStatus()
		case "updates.check":
			return service.CheckUpdate(ctx)
		default:
			return service.InstallUpdate(ctx)
		}
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
