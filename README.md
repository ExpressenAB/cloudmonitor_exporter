# Cloudmonitor exporter
Prometheus exporter for Akamai Cloudmonitor statistics.
Retrieves json data from akamai cloudmonitor (https://control.akamai.com/dl/customers/ALTA/Cloud-Monitor-Implementation.pdf) on collector.endpoint and provides metrics on metrics.endpoint

## Get it
The latest version is 0.1.0 and all releases can be found under [Releases](https://github.com/ExpressenAB/cloudmonitor_exporter/releases).

## Usage
Example: 
```
./cloudmonitor_exporter
```

### Flags
Flag | Description | Default
-----|-------------|---------
-exporter.address | Exporter bind address:port | :9143
-exporter.namespace | The namespace used in prometheus labels | cloudmonitor
-metrics.endpoint | Metrics endpoint | /metrics
-collector.endpoint | Collector endpoint | /collector
-collector.accesslog | File to store accesslogs to | "" off
