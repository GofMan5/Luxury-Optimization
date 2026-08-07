package app

import (
	"context"
	"encoding/json"

	"github.com/GofMan5/Luxury-Optimization/internal/features/benchmark"
	"github.com/GofMan5/Luxury-Optimization/internal/features/cleanup"
	"github.com/GofMan5/Luxury-Optimization/internal/features/gaming"
	"github.com/GofMan5/Luxury-Optimization/internal/features/network"
	"github.com/GofMan5/Luxury-Optimization/internal/features/optimization"
	"github.com/GofMan5/Luxury-Optimization/internal/features/services"
	"github.com/GofMan5/Luxury-Optimization/internal/features/startup"
	"github.com/GofMan5/Luxury-Optimization/internal/features/system"
	"github.com/GofMan5/Luxury-Optimization/internal/features/updates"
	"github.com/GofMan5/Luxury-Optimization/internal/optimizer"
	"github.com/GofMan5/Luxury-Optimization/internal/protocol"
)

var AllowedMethods = []string{
	"system.handshake",
	"system.cancel",
	"system.shutdown",
	"optimization.audit",
	"optimization.plan",
	"optimization.apply",
	"optimization.restore",
	"optimization.apply_tweak",
	"optimization.restore_tweak",
	"optimization.checkpoint_status",
	"optimization.create_checkpoint",
	"backups.list",
	"restore.system_points",
	"restore.open_system",
	"startup.list",
	"startup.set",
	"services.list",
	"services.set",
	"network.interfaces",
	"network.test",
	"benchmark.compare",
	"gaming.scan",
	"gaming.saved",
	"gaming.save",
	"gaming.remove",
	"gaming.launch",
	"gaming.history",
	"gaming.attach_benchmark",
	"cleanup.run",
	"updates.status",
}

type Application struct {
	optimizer *optimizer.Service
}

func New() *Application { return &Application{optimizer: optimizer.NewService()} }

func (application *Application) Handle(ctx context.Context, method string, payload json.RawMessage) (any, error) {
	switch method {
	case "system.handshake":
		return system.Handle(method, payload, AllowedMethods)
	case "optimization.audit", "optimization.plan", "optimization.apply", "optimization.restore", "optimization.apply_tweak", "optimization.restore_tweak", "optimization.checkpoint_status", "optimization.create_checkpoint", "backups.list", "restore.system_points", "restore.open_system":
		return optimization.Handle(ctx, application.optimizer, method, payload)
	case "startup.list", "startup.set":
		return startup.Handle(ctx, application.optimizer, method, payload)
	case "services.list", "services.set":
		return services.Handle(ctx, application.optimizer, method, payload)
	case "network.interfaces", "network.test":
		return network.Handle(ctx, application.optimizer, method, payload)
	case "benchmark.compare":
		return benchmark.Handle(ctx, application.optimizer, method, payload)
	case "gaming.scan", "gaming.saved", "gaming.save", "gaming.remove", "gaming.launch", "gaming.history", "gaming.attach_benchmark":
		return gaming.Handle(ctx, application.optimizer, method, payload)
	case "cleanup.run":
		return cleanup.Handle(ctx, application.optimizer, method, payload)
	case "updates.status":
		return updates.Handle(ctx, application.optimizer, method, payload)
	default:
		return nil, protocol.ErrMethodNotFound
	}
}
