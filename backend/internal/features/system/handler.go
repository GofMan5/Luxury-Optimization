package system

import (
	"encoding/json"
	"runtime"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type Handshake struct {
	Product  string   `json:"product"`
	Version  string   `json:"version"`
	Protocol int      `json:"protocol"`
	OS       string   `json:"os"`
	Arch     string   `json:"arch"`
	Methods  []string `json:"methods"`
}

func Handle(method string, payload json.RawMessage, methods []string) (any, error) {
	if method != "system.handshake" {
		return nil, protocol.ErrMethodNotFound
	}
	if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
		return nil, err
	}
	return Handshake{Product: "Luxury Optimization", Version: optimizer.ProductVersion(), Protocol: protocol.Version, OS: runtime.GOOS, Arch: runtime.GOARCH, Methods: methods}, nil
}
