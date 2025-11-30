package daemon

import (
	"appmon/internal/config"
	"appmon/internal/ipc"
	"appmon/internal/monitor"
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func main() {
	// set path to user home dir
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot get home dir: %v", err)
	}

	cfgPath := filepath.Join(home, ".config", "appmon", "config.yaml")
	socketPath := filepath.Join(home, ".config", "appmon", "appmon.sock")

	// try load config; if not then create default
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("config load failed (%v), creating default config", err)
		cfg = &config.AppConfig{Apps: []config.App{
			config.App{Username: "Firefox", SystemName: "firefox", Limit: 60},
			config.App{Username: "VScode", SystemName: "code", Limit: 60},
		}}
		// try to save
		if saveErr := config.Save(cfgPath, cfg); saveErr != nil {
			log.Printf("failed to save default config: %v", saveErr)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cancelIPC := make(chan struct{})

	m := monitor.NewMonitor(cfg.Apps)

	var wg sync.WaitGroup

	// starting monitor
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.Run(ctx)
	}()

	// starting IPC
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := ipc.StartServer(cancelIPC, socketPath, func(req ipc.Request) ipc.Response {
			switch req.Cmd {
			case "getLimits":
				return ipc.Response{Success: true, Data: map[string]any{"limits": cfg.Apps}}
			case "setLimit":
				cfg.Apps = append(cfg.Apps, config.App{
					Username:   req.Name,
					SystemName: req.App,
					Limit:      req.Limit})
				m.SetLimit(req.App, req.Limit)
				_ = config.Save(cfgPath, cfg)
				return ipc.Response{Success: true}
			case "getTimers":
				return ipc.Response{Success: true, Data: map[string]any{"timers": m.GetTimers()}}
			default:
				return ipc.Response{Success: false}
			}
		})
		if err != nil {
			log.Printf("ipc server stopped: %v", err)
		}
	}()

	// waiting for stopping
	<-ctx.Done()
	log.Printf("shutdown requested, stopping...")

	// stop IPC
	close(cancelIPC)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Timeout на graceful shutdown
	select {
	case <-done:
		log.Printf("all goroutines stopped")
	case <-time.After(5 * time.Second):
		log.Printf("timeout waiting for goroutines, exiting")
	}

	// Удаляем socket файл если остался
	_ = os.Remove(socketPath)
	log.Printf("daemon exited")
}
