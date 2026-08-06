package optimization

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type profileRequest struct {
	Profile string `json:"profile"`
}

type restoreRequest struct {
	ID string `json:"id,omitempty"`
}

type checkpointRequest struct {
	Profile string `json:"profile"`
}

type tweakRequest struct {
	ID string `json:"id"`
}

func Handle(_ context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "optimization.audit":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.Audit(), nil
	case "optimization.plan", "optimization.apply":
		request, err := protocol.DecodePayload[profileRequest](payload)
		if err != nil {
			return nil, err
		}
		if method == "optimization.plan" {
			return service.Plan(request.Profile)
		}
		return service.ApplyProfile(request.Profile)
	case "optimization.restore":
		request, err := protocol.DecodePayload[restoreRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.Restore(request.ID)
	case "optimization.apply_tweak", "optimization.restore_tweak":
		request, err := protocol.DecodePayload[tweakRequest](payload)
		if err != nil {
			return nil, err
		}
		if method == "optimization.apply_tweak" {
			return service.ApplyTweak(request.ID)
		}
		return service.RestoreTweak(request.ID)
	case "backups.list":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.Backups()
	case "restore.system_points":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.SystemRestorePoints()
	case "restore.open_system":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.OpenSystemRestore()
	case "optimization.checkpoint_status", "optimization.create_checkpoint":
		request, err := protocol.DecodePayload[checkpointRequest](payload)
		if err != nil {
			return nil, err
		}
		if method == "optimization.checkpoint_status" {
			return service.CheckpointStatus(request.Profile)
		}
		return service.CreateCheckpoint(request.Profile)
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
