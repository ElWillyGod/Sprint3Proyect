package main

import (
	"sync"
	"time"
)

type Metrics struct {
	mu                sync.RWMutex
	requestCount      int64
	requestsPerSecond float64
	avgResponseTime   time.Duration
	windowStart       time.Time
	windowRequests    int64
	cacheHitsL1       int64
	cacheHitsL2       int64
}

var appMetrics = &Metrics{
	windowStart: time.Now(),
}

func (m *Metrics) IncrementRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.requestCount++

	// Ventana deslizante de 1 minuto
	if now.Sub(m.windowStart) >= time.Minute {
		m.requestsPerSecond = float64(m.windowRequests) / 60.0
		m.windowRequests = 0
		m.windowStart = now
	}
	m.windowRequests++
}

func (m *Metrics) UpdateResponseTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Promedio móvil simple
	if m.avgResponseTime == 0 {
		m.avgResponseTime = duration
	} else {
		m.avgResponseTime = (m.avgResponseTime + duration) / 2
	}
}

func (m *Metrics) IncrementCacheHit(cacheType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch cacheType {
	case "l1":
		m.cacheHitsL1++
	case "l2":
		m.cacheHitsL2++
	}
}

func (m *Metrics) GetCacheHits(cacheType string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch cacheType {
	case "l1":
		return m.cacheHitsL1
	case "l2":
		return m.cacheHitsL2
	default:
		return 0
	}
}

func (m *Metrics) GetMetrics() (float64, time.Duration, int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestsPerSecond, m.avgResponseTime, m.requestCount
}

func (m *Metrics) ShouldScaleUp() bool {
	rps, avgTime, _ := m.GetMetrics()
	// Escalar si > 10 RPS o tiempo promedio > 100ms
	return rps > 10 || avgTime > 100*time.Millisecond
}

func (m *Metrics) ShouldScaleDown() bool {
	rps, avgTime, _ := m.GetMetrics()
	// Reducir si < 2 RPS y tiempo promedio < 50ms
	return rps < 2 && avgTime < 50*time.Millisecond
}
