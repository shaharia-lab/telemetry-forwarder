# OpenTelemetry to Honeycomb Forwarder

A lightweight service that forwards OpenTelemetry events to Honeycomb.io.

## Overview

This service acts as a bridge between applications sending OpenTelemetry-formatted telemetry data and the Honeycomb.io
observability platform. It receives OpenTelemetry events via an HTTP endpoint and forwards them to Honeycomb's API.

## Features

- Accepts OpenTelemetry-formatted events via HTTP
- Transforms events to Honeycomb's format
- CORS support for cross-origin requests
- Configurable via environment variables

## Configuration

The service is configured using environment variables:

| Variable            | Description                             | Default             |
|---------------------|-----------------------------------------|---------------------|
| `PORT`              | The port the server will listen on      | `8080`              |
| `HONEYCOMB_API_KEY` | Your Honeycomb API key                  | (required)          |
| `HONEYCOMB_DATASET` | The Honeycomb dataset to send events to | `cli-telemetry`     |
| `HONEYCOMB_API_URL` | Custom Honeycomb API URL (optional)     | (Honeycomb default) |

## Getting Started

### Prerequisites

- Go 1.x

### Running the Service

1. Set required environment variables:

```bash
export HONEYCOMB_API_KEY=your-api-key
```

2. Build and run the service:

```bash
go build
./otel-honeycomb-forwarder
```

## API

### POST /telemetry/event

Accepts OpenTelemetry events and forwards them to Honeycomb.

**Request Body**: An OpenTelemetry event with the following structure:

```json
{
  "name": "event_name",
  "timeUnixNano": 1639083272000000000,
  "traceId": "trace-id",
  "spanId": "span-id",
  "severityText": "INFO",
  "severityNumber": 9,
  "body": "Event message or data",
  "attributes": {
    "key1": "value1",
    "key2": "value2"
  },
  "resource": {
    "service.name": "my-service"
  }
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.