package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	autoheal "github.com/openshift/autoheal/pkg/apis/autoheal/v1alpha2"
)

var (
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
	prometheus.MustRegister(actionsInitiated)
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
