FROM alpine:latest
ARG VERSION=0.0.0
COPY "build/cloudmonitor_exporter_${VERSION}_linux_amd64/cloudmonitor_exporter" "/app/cloudmonitor_exporter"
EXPOSE 9143
CMD ["/app/cloudmonitor_exporter"]
