package network

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
)

func TestDiagnosticHandlersRejectUnknownAndUnboundedPayloads(t *testing.T) {
	service := optimizer.NewService()
	for method, payload := range map[string]string{
		"network.udp":         `{"address":"1.1.1.1:53","count":10,"timeout_ms":2000,"extra":true}`,
		"network.bufferbloat": `{"probe_address":"1.1.1.1:443","download_url":"http://example.com/down","upload_url":"https://example.com/up","duration_ms":3000,"streams":1}`,
	} {
		if _, err := Handle(context.Background(), service, method, json.RawMessage(payload)); err == nil {
			t.Fatalf("%s payload accepted", method)
		}
	}
}
