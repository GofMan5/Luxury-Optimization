package advisor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
)

func TestBackgroundHandlerRejectsUnboundedAndUnknownPayloads(t *testing.T) {
	service := optimizer.NewService()
	for _, payload := range []string{`{"sample_ms":499}`, `{"sample_ms":5001}`, `{"sample_ms":1000,"extra":true}`} {
		if _, err := Handle(context.Background(), service, "advisor.background", json.RawMessage(payload)); err == nil {
			t.Fatalf("payload accepted: %s", payload)
		}
	}
}
