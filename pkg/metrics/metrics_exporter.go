package metrics

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

// Handle /metrics requsts, retrun a list of all exported metrics
//
func Handler() http.Handler {
	return promhttp.Handler()
}

// Init autoheal prometheus exported metrics
//
func InitExportedMetrics() {
	prometheus.MustRegister(actionsRequested, actionsLaunched)
}

func ActionStarted(
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

func ActionCompleted(
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

func ActionRequested(actionType, rule, alert string) {
	actionsRequested.With(
		map[string]string{
			"type":  actionType,
			"rule":  rule,
			"alert": alert,
		},
	).Inc()
}
