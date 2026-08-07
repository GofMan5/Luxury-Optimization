package network

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type testRequest struct {
	Address   string `json:"address"`
	Count     int    `json:"count"`
	TimeoutMS int    `json:"timeout_ms"`
}

type udpRequest struct {
	Address   string `json:"address"`
	Count     int    `json:"count"`
	TimeoutMS int    `json:"timeout_ms"`
}

func Handle(ctx context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "network.interfaces":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.NetworkInterfaces()
	case "network.test":
		request, err := protocol.DecodePayload[testRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.NetworkTestContext(ctx, request.Address, request.Count, request.TimeoutMS)
	case "network.udp":
		request, err := protocol.DecodePayload[udpRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.NetworkUDP(ctx, request.Address, request.Count, request.TimeoutMS)
	case "network.bufferbloat":
		request, err := protocol.DecodePayload[optimizer.BufferbloatRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.NetworkBufferbloat(ctx, request)
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
