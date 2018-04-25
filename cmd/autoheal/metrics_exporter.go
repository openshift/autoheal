package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	actionsRequested = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoheal_actions_requested_total",
			Help: "Number of requested healing actions(including rate limited)",
		},
		[]string{"type", "rule", "alert"},
	)
	actionsLaunched = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "autoheal_actions_launched",
			Help: "Number of launched healing actions(including completed)",
		},
		[]string{"type", "template", "rule", "status"},
	)
)

func (h *Healer) metricsHandler() http.Handler {
	return promhttp.Handler()
}

func (h *Healer) initExportedMetrics() {
	prometheus.MustRegister(actionsRequested, actionsLaunched)
}

func (h *Healer) actionStarted(
	actionType,
	templateName,
	ruleName string,
) {
	actionsLaunched.With(
		map[string]string{
			"type":     actionType,
			"template": templateName,
			"rule":     ruleName,
			"status":   "running",
		},
	).Inc()
}

func (h *Healer) actionCompleted(
	actionType,
	templateName,
	ruleName string,
) {
	actionsLaunched.With(
		map[string]string{
			"type":     actionType,
			"template": templateName,
			"rule":     ruleName,
			"status":   "running",
		},
	).Dec()
	actionsLaunched.With(
		map[string]string{
			"type":     actionType,
			"template": templateName,
			"rule":     ruleName,
			"status":   "completed",
		},
	).Inc()
}

func (h *Healer) actionRequested(actionType, rule, alert string) {
	actionsRequested.With(
		map[string]string{
			"type":  actionType,
			"rule":  rule,
			"alert": alert,
		},
	).Inc()
}
