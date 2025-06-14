# Metrics Example

This example demonstrates how to use mahakam's metrics extension.

## Usage

To run the example, you can use docker compose:

```bash
docker compose up
```

You can then access the server at `http://localhost:8080` and the grafana dashboard at `http://localhost:3000`.

## Metrics

The metrics extension provides the following metrics:

- `http_requests_total`: The total number of processed events.
- `http_request_duration_seconds`: The duration of the request.
- `http_request_size_bytes`: The size of HTTP requests in bytes.
- `http_active_connections`: Number of active HTTP connections.
- `http_concurrent_requests`: Number of concurrent HTTP requests being processed.
- `http_errors_total`: Total number of HTTP errors.
- `http_server_uptime_seconds`: Total uptime of the HTTP server in seconds.