# This dockerfile is only used to build the backend image for the application using goreleaser.
FROM golang:1.24-alpine
COPY telemetry-forwarder /app/telemetry-forwarder
ENTRYPOINT ["/app/telemetry-forwarder"]
