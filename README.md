# Wind Farm Turbine Monitor Service

A Go service for wind-farm operations. It ingests SCADA telemetry, validates and
normalizes samples, evaluates alarm rules, drives a fault state machine,
dispatches maintenance work orders, computes power-curve analytics and produces
site reports. All domain state is memory-backed; a small HTTP API and a status
page are served by the same process.

## Structure

- `cmd/server`: HTTP service entrypoint and dependency wiring.
- `internal/turbine`: fleet registry and turbine configuration.
- `internal/telemetry`: SCADA sample validation, normalization, ingestion and storage.
- `internal/ruleengine`: threshold alarm rules and evaluation.
- `internal/fault`: fault state machine and lifecycle.
- `internal/alarm`: alarm dedup, dispatch, escalation and notification.
- `internal/workorder`: maintenance scheduling and crew assignment.
- `internal/analytics`: sliding windows, power curve and aggregation.
- `internal/report`: site report building.
- `internal/audit`: event stream, checkpoint and delivery.
- `internal/gateway`: HTTP routing and handlers.
- `web`: a small operations status page served by the Go process.

## Run

```bash
go run ./cmd/server
```

The service listens on `PORT` (default `18080`). Health is at `/health`; a
summary is at `/api/summary`; telemetry is ingested at `POST /api/telemetry`.

## Test

```bash
go test ./...
go test -race ./...
```

## Environment

- `PORT`: HTTP port, defaults to `18080`.
- `SITE_CODE`: logical site identifier, defaults to `NORTH-PLAINS`.
- `DEFAULT_CUT_IN` / `DEFAULT_CUT_OUT`: fleet default wind-speed bounds.
