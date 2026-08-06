package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type server struct {
	application *Application
	output      io.Writer
	writeMu     sync.Mutex
	callMu      sync.Mutex
	calls       map[string]context.CancelFunc
	wait        sync.WaitGroup
}

type cancelRequest struct {
	ID string `json:"id"`
}

func Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	ctx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()
	server := &server{application: New(), output: output, calls: make(map[string]context.CancelFunc)}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 8<<10), protocol.MaxRequestBytes)
	for scanner.Scan() {
		command, err := protocol.DecodeCommand(scanner.Bytes())
		if err != nil {
			return err
		}
		switch command.Method {
		case "system.handshake":
			payload, err := server.application.Handle(ctx, command.Method, command.Payload)
			if err != nil {
				server.writeError(command.ID, "invalid_request", err)
			} else {
				server.writeSuccess(command.ID, payload)
			}
			continue
		case "system.shutdown":
			if _, err := protocol.DecodePayload[struct{}](command.Payload); err != nil {
				server.writeError(command.ID, "invalid_request", err)
				continue
			}
			cancelAll()
			server.cancelCalls()
			server.waitCalls(2 * time.Second)
			server.writeSuccess(command.ID, map[string]bool{"stopping": true})
			return nil
		case "system.cancel":
			request, err := protocol.DecodePayload[cancelRequest](command.Payload)
			if err != nil {
				server.writeError(command.ID, "invalid_request", err)
				continue
			}
			server.writeSuccess(command.ID, map[string]bool{"cancelled": server.cancelCall(request.ID)})
			continue
		}
		if !server.startCall(ctx, command) {
			server.writeError(command.ID, "duplicate_request", errors.New("request ID is already active"))
		}
	}
	cancelAll()
	server.cancelCalls()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("sidecar input: %w", err)
	}
	return nil
}

func (server *server) startCall(parent context.Context, command protocol.Command) bool {
	ctx, cancel := context.WithCancel(parent)
	server.callMu.Lock()
	if _, exists := server.calls[command.ID]; exists {
		server.callMu.Unlock()
		cancel()
		return false
	}
	server.calls[command.ID] = cancel
	server.wait.Add(1)
	server.callMu.Unlock()
	go func() {
		defer func() {
			server.wait.Done()
			server.callMu.Lock()
			delete(server.calls, command.ID)
			server.callMu.Unlock()
			cancel()
		}()
		payload, err := server.application.Handle(ctx, command.Method, command.Payload)
		if ctx.Err() != nil {
			server.writeError(command.ID, "cancelled", ctx.Err())
			return
		}
		if err != nil {
			code := "operation_failed"
			if errors.Is(err, protocol.ErrMethodNotFound) {
				code = "method_not_found"
			} else if strings.HasPrefix(err.Error(), "invalid payload:") {
				code = "invalid_request"
			}
			server.writeError(command.ID, code, err)
			return
		}
		server.writeSuccess(command.ID, payload)
	}()
	return true
}

func (server *server) waitCalls(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		server.wait.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func (server *server) cancelCall(id string) bool {
	server.callMu.Lock()
	cancel, exists := server.calls[id]
	server.callMu.Unlock()
	if exists {
		cancel()
	}
	return exists
}

func (server *server) cancelCalls() {
	server.callMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(server.calls))
	for _, cancel := range server.calls {
		cancels = append(cancels, cancel)
	}
	server.callMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (server *server) writeSuccess(id string, payload any) {
	server.write(protocol.Result{Version: protocol.Version, ID: id, Type: "result", OK: true, Payload: payload})
}

func (server *server) writeError(id, code string, err error) {
	server.write(protocol.Result{Version: protocol.Version, ID: id, Type: "result", OK: false, Error: &protocol.ErrorPayload{Code: code, Message: safeMessage(err)}})
}

func (server *server) write(result protocol.Result) {
	data, err := protocol.MarshalResult(result)
	if err != nil {
		data, _ = json.Marshal(protocol.Result{Version: protocol.Version, ID: result.ID, Type: "result", OK: false, Error: &protocol.ErrorPayload{Code: "result_too_large", Message: "Result exceeds the protocol limit."}})
	}
	server.writeMu.Lock()
	defer server.writeMu.Unlock()
	_, _ = server.output.Write(append(data, '\n'))
}

func safeMessage(err error) string {
	if err == nil {
		return "Unknown error."
	}
	var builder strings.Builder
	for _, value := range err.Error() {
		if unicode.IsControl(value) {
			continue
		}
		builder.WriteRune(value)
		if builder.Len() >= 1000 {
			break
		}
	}
	if builder.Len() == 0 {
		return "Operation failed."
	}
	return builder.String()
}
