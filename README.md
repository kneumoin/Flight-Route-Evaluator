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

## Travelpayouts Data API

This project supports Travelpayouts / Aviasales **Data API** as a cached price provider (`travelpayouts_data`).
It is **not** real-time search and does not guarantee live availability, baggage, or exact itinerary data.

Set token via environment variable (never commit tokens to git):

```bash
export TRAVELPAYOUTS_TOKEN="..."
```

Run:

```bash
TRAVELPAYOUTS_TOKEN=... go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out \
  --provider travelpayouts_data \
  --verbose
```

Optional CLI override (also never logged):

```bash
go run ./cmd/flight-routes ... --travelpayouts-token "..."
```

`AVIASALES_TOKEN` is accepted as an alias for `TRAVELPAYOUTS_TOKEN`.

## Experimental Aviasales browser provider

**Local-only, disabled by default, not used in CI or golden tests.**

The `aviasales_browser` provider opens Aviasales public search pages in a **visible local browser** (Chrome/Chromium via [chromedp](https://github.com/chromedp/chromedp)) and extracts visible offers. Use only for personal one-time ticket research on your machine.

- No captcha bypass, no anti-bot evasion, no login automation
- Low request rate (default: 1 page per minute)
- Results cached under `.cache/aviasales_browser/`
- UI selectors may break when Aviasales changes layout

```bash
go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out \
  --provider aviasales_browser \
  --browser-headful=true \
  --browser-rate-limit=1m \
  --verbose
```

Flags: `--browser-cache-only`, `--browser-timeout=120s`

Requires Google Chrome or Chromium installed locally.

## Other providers (optional)

```bash
export AVIASALES_TOKEN="..."   # legacy aviasales provider id
export KIWI_API_KEY="..."      # optional Kiwi stub

go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out
```
