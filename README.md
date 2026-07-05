# Flight Route Evaluator

Config-driven CLI to evaluate predefined flight route branches. Origin, destination, and candidate routes are defined in a YAML config.

## Prerequisites

- Go 1.22+
- No database setup

## Quick start (mock / no API keys)

```bash
go test ./...
go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out \
  --provider mock
open ./out/report.html
```

## Production run (real providers)

```bash
export AVIASALES_TOKEN="..."
export KIWI_API_KEY="..."   # optional

go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out
```
