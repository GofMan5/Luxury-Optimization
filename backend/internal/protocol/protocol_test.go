package protocol

import "testing"

func TestDecodeCommandRejectsUnknownFieldsAndBadIDs(t *testing.T) {
	for _, frame := range []string{
		`{"v":1,"id":"bad id","type":"command","method":"system.handshake"}`,
		`{"v":1,"id":"x","type":"command","method":"system.handshake","extra":true}`,
		`{"v":2,"id":"x","type":"command","method":"system.handshake"}`,
	} {
		if _, err := DecodeCommand([]byte(frame)); err == nil {
			t.Fatalf("invalid frame accepted: %s", frame)
		}
	}
}

func TestDecodeCommandAcceptsBoundedEnvelope(t *testing.T) {
	command, err := DecodeCommand([]byte(`{"v":1,"id":"request-1","type":"command","method":"system.handshake","payload":{}}`))
	if err != nil || command.Method != "system.handshake" {
		t.Fatalf("command=%#v err=%v", command, err)
	}
}
