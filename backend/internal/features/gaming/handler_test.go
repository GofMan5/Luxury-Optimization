package gaming

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
)

func TestGamingHandlerRejectsInvalidIDsAndUnknownPayloadFields(t *testing.T) {
	service := optimizer.NewService()
	if _, err := Handle(context.Background(), service, "gaming.history", json.RawMessage(`{"id":"bad"}`)); err == nil {
		t.Fatal("invalid game ID accepted")
	}
	if _, err := Handle(context.Background(), service, "gaming.remove", json.RawMessage(`{"id":"0123456789ab","extra":true}`)); err == nil {
		t.Fatal("unknown payload field accepted")
	}
}
