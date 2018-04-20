/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package receiver

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
