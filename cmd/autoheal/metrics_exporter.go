package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	autoheal "github.com/openshift/autoheal/pkg/apis/autoheal/v1alpha2"
)

var (
	actionsRequested = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoheal_actions_requested_total",
			Help: "Number of requested healing actions(including rate limited)",
		},
		[]string{"type", "rule", "alert"},
	)
	actionsInitiated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoheal_actions_initiated_total",
			Help: "Number of initiated healing actions",
		},
		[]string{"type", "template", "rule"},
	)
)

func (h *Healer) metricsHandler() http.Handler {
	return promhttp.Handler()
}

func (h *Healer) initExportedMetrics() {
	prometheus.MustRegister(actionsRequested, actionsInitiated)
}

func (h *Healer) incrementAwxActions(
	action *autoheal.AWXJobAction,
	ruleName string,
) {
	actionsInitiated.With(
		map[string]string{
			"type":     "awxJob",
			"template": action.Template,
			"rule":     ruleName,
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
