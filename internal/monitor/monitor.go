package monitor

import (
	"appmon/internal/config"
	"appmon/internal/notify"
	"context"
	"sync"
	"time"
)

type ActiveApp struct {
	Name string
	PID  int
}

type Monitor struct {
	timers map[string]int
	limits map[string]int

	mu sync.Mutex
}

func NewMonitor(limits []config.App) *Monitor {
	lm := make(map[string]int)
	for _, app := range limits {
		lm[app.SystemName] = app.Limit
	}
	return &Monitor{
		timers: make(map[string]int),
		limits: lm,
	}
}

func getActiveApp() ActiveApp {
	return ActiveApp{Name: "firefox", PID: 1}
}

func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app := getActiveApp()
			m.mu.Lock()
			limitMinutes, ok := m.limits[app.Name]
			if !ok || limitMinutes <= 0 {
				m.mu.Unlock()
				continue
			}

			m.timers[app.Name]++
			if m.timers[app.Name] >= limitMinutes*60 {
				notify.Send(app.Name)
				m.timers[app.Name] = 0
			}
			m.mu.Unlock()
		}
	}
}

func (m *Monitor) GetTimers() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int, len(m.timers))
	for k, v := range m.timers {
		out[k] = v
	}
	return out
}

func (m *Monitor) SetLimit(app string, minutes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limits[app] = minutes
	m.timers[app] = 0
}
