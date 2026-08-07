package storage

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

type testRequest struct {
	Path    string `json:"path"`
	SizeMB  int    `json:"size_mb"`
	BlockKB int    `json:"block_kb"`
}

type scanStartRequest struct {
	Path          string `json:"path"`
	ParentScanID  string `json:"parent_scan_id"`
	NodeID        string `json:"node_id"`
	RefreshScanID string `json:"refresh_scan_id"`
}

type scanIDRequest struct {
	ScanID string `json:"scan_id"`
}

type deletePreviewRequest struct {
	ScanID string `json:"scan_id"`
	NodeID string `json:"node_id"`
}

type deleteConfirmRequest struct {
	ScanID            string `json:"scan_id"`
	ConfirmationToken string `json:"confirmation_token"`
}

func Handle(ctx context.Context, service *optimizer.Service, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "storage.volumes":
		if _, err := protocol.DecodePayload[struct{}](payload); err != nil {
			return nil, err
		}
		return service.StorageVolumes()
	case "storage.test":
		request, err := protocol.DecodePayload[testRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.StorageTest(ctx, request.Path, request.SizeMB, request.BlockKB)
	case "storage.scan.start":
		request, err := protocol.DecodePayload[scanStartRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.StartStorageScan(request.Path, request.ParentScanID, request.NodeID, request.RefreshScanID)
	case "storage.scan.status":
		request, err := protocol.DecodePayload[scanIDRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.StorageScanStatus(request.ScanID)
	case "storage.scan.cancel":
		request, err := protocol.DecodePayload[scanIDRequest](payload)
		if err != nil {
			return nil, err
		}
		cancelled, err := service.CancelStorageScan(request.ScanID)
		return map[string]bool{"cancelled": cancelled}, err
	case "storage.delete.preview":
		request, err := protocol.DecodePayload[deletePreviewRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.PreviewStorageDelete(request.ScanID, request.NodeID)
	case "storage.delete.confirm":
		request, err := protocol.DecodePayload[deleteConfirmRequest](payload)
		if err != nil {
			return nil, err
		}
		return service.ConfirmStorageDelete(request.ScanID, request.ConfirmationToken)
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
