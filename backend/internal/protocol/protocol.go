package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const (
	Version         = 1
	MaxRequestBytes = 256 << 10
	MaxResultBytes  = 1 << 20
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

var ErrMethodNotFound = errors.New("method not found")

type Command struct {
	Version int             `json:"v"`
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Method  string          `json:"method"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Result struct {
	Version int           `json:"v"`
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	OK      bool          `json:"ok"`
	Payload any           `json:"payload,omitempty"`
	Error   *ErrorPayload `json:"error,omitempty"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func DecodeCommand(data []byte) (Command, error) {
	var command Command
	if len(data) == 0 || len(data) > MaxRequestBytes {
		return command, errors.New("invalid protocol frame size")
	}
	if err := decodeStrict(data, &command); err != nil {
		return command, fmt.Errorf("invalid protocol frame: %w", err)
	}
	if command.Version != Version || command.Type != "command" || !requestIDPattern.MatchString(command.ID) || command.Method == "" {
		return command, errors.New("invalid protocol envelope")
	}
	return command, nil
}

func DecodePayload[T any](data json.RawMessage) (T, error) {
	var value T
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		data = []byte("{}")
	}
	if err := decodeStrict(data, &value); err != nil {
		return value, fmt.Errorf("invalid payload: %w", err)
	}
	return value, nil
}

func MarshalResult(result Result) ([]byte, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxResultBytes {
		return nil, errors.New("result exceeds protocol limit")
	}
	return data, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
}
