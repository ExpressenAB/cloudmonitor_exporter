FROM alpine:latest

ARG version=0.0.0

COPY build/cloudmonitor_exporter_${version}_linux_amd64/cloudmonitor_exporter /user/bin

ENTRYPOINT /usr/bin/cloudmonitor_exporter

EXPOSE 9143