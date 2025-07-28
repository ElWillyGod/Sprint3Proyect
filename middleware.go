package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// Middleware para capturar métricas
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		appMetrics.IncrementRequests()

		// Ejecutar el handler
		next(w, r)

		// Actualizar tiempo de respuesta
		duration := time.Since(start)
		appMetrics.UpdateResponseTime(duration)
	}
}

// Endpoint para exponer métricas en formato JSON
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	rps, avgTime, total := appMetrics.GetMetrics()

	metrics := map[string]interface{}{
		"total_requests":       total,
		"requests_per_second":  rps,
		"avg_response_time_ms": avgTime.Milliseconds(),
		"cache_hits": map[string]interface{}{
			"l1_cache": appMetrics.GetCacheHits("l1"),
			"l2_cache": appMetrics.GetCacheHits("l2"),
		},
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
