### Basic setup with docker-compose

Docker-compose can be used to setup an small stack with prometheus/grafana/haproxy/cloudmonitor_exporter.

## Instructions
* Make sure docker-compose is installed ([instructions](https://docs.docker.com/compose/install/))
* Clone repository

`git clone git@github.com:ExpressenAB/cloudmonitor_exporter.git`
* Create self-signed certificate by running setup.sh
```
> cd cloudmonitor_exporter/setup
> ./setup.sh
This will generate a self-signed certificate to use with cloudmonitor_exporter
Enter companyname for certificate:
......
```
* Start containers by running `docker-compose up`

Now we have 4 docker containers running with:
* cloudmonitor_exporter listening on :9143
* haproxy listening on 443 with self-signed certificate for ssl termination
* prometheus scraping localhost:9143
* grafana using prometheus datasource from localhost:9090

## Test stack

The stack can now be tested by running localtest.sh under `tests`
```
cd tests
./localtest.sh
```

Now visit local [grafana](http://localhost:3000) instance and login with admin/admin
