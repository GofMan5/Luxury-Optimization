package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestHandshakeAndShutdownRoundTrip(t *testing.T) {
	input := strings.NewReader("{\"v\":1,\"id\":\"hello\",\"type\":\"command\",\"method\":\"system.handshake\",\"payload\":{}}\n{\"v\":1,\"id\":\"bye\",\"type\":\"command\",\"method\":\"system.shutdown\",\"payload\":{}}\n")
	var output strings.Builder
	if err := Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected frames: %q", output.String())
	}
	for _, line := range lines {
		var frame struct {
			OK bool `json:"ok"`
		}
		if err := json.Unmarshal([]byte(line), &frame); err != nil || !frame.OK {
			t.Fatalf("frame=%q err=%v", line, err)
		}
	}
}
