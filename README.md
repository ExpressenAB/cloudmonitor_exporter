# Cloudmonitor exporter
Prometheus exporter for Akamai Cloudmonitor statistics.
Retrieves json data from akamai cloudmonitor [Here](https://control.akamai.com/dl/customers/ALTA/Cloud-Monitor-Implementation.pdf) on `collector.endpoint` and provides metrics on `metrics.endpoint`

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

## Description
Akamai cloud monitor will aggregate client request/responses as json data and send them to specified endpoint.
Detailed information can be found [Here](https://control.akamai.com/dl/customers/ALTA/Cloud-Monitor-Implementation.pdf)

This exporter will gather this data and expose an `/metrics` endpoint to prometheus.

To be able to retrieve cloudmonitor data to the running exporter, you need to to create an cloudmonitor property, that other properties will send loglines to.

For example:

![alt text](docs/akamai_config.png "Akamai config")

Then it's enough to add the following behavior to your properties.

![alt text](docs/akamai_behavior.png "Akamai behavior")

When properties are active and data is retrieved we will be able to query prometheus.

![alt text](docs/prometheus.png "Prometheus")

An example of metrics used on grafana dashboard
![alt text](docs/grafana.png "Prometheus")






