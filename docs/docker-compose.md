## Basic setup with docker-compose
To setup an small test environment using docker-compose:
* Make sure docker-compose is installed ([instructions](https://docs.docker.com/compose/install/))
* Clone this repository

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
