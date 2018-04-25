# Metrics

autoheal uses [Prometheus](https://prometheus.io/) for metrics reporting. The metrics can be used for real-time monitoring and debugging. The auto-heal service does not persist its metrics; the metrics will be reset upon restart.

The simplest way to see the available metrics is to cURL the metrics endpoint `/metrics`. The format is described [here](http://prometheus.io/docs/instrumenting/exposition_formats/).

Follow the [Prometheus getting started doc](https://prometheus.io/docs/prometheus/latest/getting_started/) to spin up a Prometheus server to collect autoheal metrics.

The naming of metrics follows the suggested [Prometheus best practices](http://prometheus.io/docs/practices/naming/). A metric name has an `autoheal`, `go` or `process` prefix as its namespace and a subsystem prefix.

## Autoheal namespace metrics

The metrics under the `autoheal` prefix expose information related to healing actions undertaken by the server

### Actions

These metrics describe the status of healing actions attempted by the server.

All these metrics are prefixed with `autoheal_actions_`

| Name             | Description                         | Type    |
|------------------|-------------------------------------|---------|
| launched         | Number of started healing actions   | Gauge   |
| requested_total  | Number of requested healing actions | Counter |

`requested_total` indicates how many healing actions were triggered by the server. An action that
was rate limited by the server is counted here as well as a heal that failed to run for some reason.
For example if autoheal failed to contact AWX for an AWX job, a heal will not start
but it will be counted as requested.

`launched` indicates how many healing actions started, partitioned by status `running`|`completed`.

## Prometheus supplied metrics

The Prometheus client library provides a number of metrics under the `go` and `process` namespaces that pertain to the entire process and the go runtime of the entire process. To find out more about these, see:
[go-collector](https://github.com/prometheus/client_golang/blob/master/prometheus/go_collector.go),
[process_collector](https://github.com/prometheus/client_golang/blob/master/prometheus/process_collector.go)
