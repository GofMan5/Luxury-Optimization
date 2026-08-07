package gaming

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type idRequest struct {
	ID string `json:"id"`
}

type attachBenchmarkRequest struct {
	GameID string                 `json:"game_id"`
	Before optimizer.BenchmarkSet `json:"before"`
	After  optimizer.BenchmarkSet `json:"after"`
}

func Handle(_ context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "gaming.scan":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.ScanGames(), nil
	case "gaming.saved":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.SavedGames()
	case "gaming.save":
		request, err := protocol.DecodePayload[optimizer.SaveGameRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.SaveGame(request)
	case "gaming.remove", "gaming.launch", "gaming.history":
		request, err := protocol.DecodePayload[idRequest](payload)
		if err != nil {
			return nil, err
		}
		switch method {
		case "gaming.remove":
			if err := service.RemoveGame(request.ID); err != nil {
				return nil, err
			}
			return optimizer.MutationResult{Changed: true, Message: "Игровой профиль удалён."}, nil
		case "gaming.launch":
			return service.LaunchGame(request.ID)
		default:
			return service.GameHistory(request.ID)
		}
	case "gaming.attach_benchmark":
		request, err := protocol.DecodePayload[attachBenchmarkRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.AttachGameBenchmark(request.GameID, request.Before, request.After)
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
