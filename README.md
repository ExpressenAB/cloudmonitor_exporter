# Cloudmonitor exporter
A Prometheus exporter for gathering [Akamai Cloudmonitor](https://www.akamai.com/us/en/solutions/intelligent-platform/cloud-monitor.jsp) statistics.

Akamai Cloudmonitor aggregates client request/responses as JSON data and send them to cloudmonitor_exporter's `collector.endpoint`. Exporter will parse this and provide metrics on the `metrics.endpoint`.

Detailed information about cloudmonitor can be found [Here](https://control.akamai.com/dl/customers/ALTA/Cloud-Monitor-Implementation.pdf)

## Get it
The latest version can be found under [Releases](https://github.com/ExpressenAB/cloudmonitor_exporter/releases).

## Usage
Example: 
```
./cloudmonitor_exporter
```

## Flags
Flag | Description | Default
-----|-------------|---------
-exporter.address | Exporter bind address:port | :9143
-exporter.namespace | The namespace used in prometheus labels | cloudmonitor
-metrics.endpoint | Metrics endpoint | /metrics
-collector.endpoint | Collector endpoint | /collector
-collector.accesslog | File to store accesslogs to | "" off

## Docker-compose
An basic stack with grafana including the template below/prometheus/haproxy/cloudmonitor_exporter can be executed with docker-compose. Instructions can be found  [Here](docs/docker-compose.md)

## Akamai setup

Information about configuration of akamai propoerties, can be found [Here](docs/akamai.md)

## Prometheus

When properties are active and data is retrieved we will be able to query prometheus.

![alt text](docs/prometheus.png "Prometheus")

## Grafana

The following [Dashboard template](setup/grafana.json), can be imported into grafana.

Example:

![alt text](docs/grafana.png "Prometheus")






