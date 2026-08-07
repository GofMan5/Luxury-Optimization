package storage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
)

func TestStorageHandlerRejectsUnknownAndUnboundedPayloads(t *testing.T) {
	service := optimizer.NewService()
	for _, payload := range []string{
		`{"path":".","size_mb":7,"block_kb":64}`,
		`{"path":".","size_mb":8,"block_kb":64,"extra":true}`,
	} {
		if _, err := Handle(context.Background(), service, "storage.test", json.RawMessage(payload)); err == nil {
			t.Fatalf("payload accepted: %s", payload)
		}
	}
	if _, err := Handle(context.Background(), service, "storage.scan.start", json.RawMessage(`{"path":"C:\\","extra":true}`)); err == nil {
		t.Fatal("unknown scan field accepted")
	}
	if _, err := Handle(context.Background(), service, "storage.scan.status", json.RawMessage(`{"scan_id":"short"}`)); err == nil {
		t.Fatal("invalid scan ID accepted")
	}
}
