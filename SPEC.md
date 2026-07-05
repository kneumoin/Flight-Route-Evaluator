# Flight Route Evaluator — Technical Specification (MVP)

> **Purpose:** Config-driven CLI tool to evaluate predefined flight route branches for expedition logistics Moscow (MOW) → Kathmandu (KTM).
>
> **This file is the immutable source of truth.** Do not change requirements during implementation unless fixing an documented contradiction (see Open Questions). Execution workflow lives in `IMPLEMENTATION_PLAN.md`; task tracking in `TASKS.yaml`; review process in `REVIEWER.md`.

---

## Problem / Context

We are planning expedition logistics from **Moscow (MOW)** to **Kathmandu (KTM)** for a trek/expedition in the **Kangchenjunga region** of Nepal.

| Parameter | Value |
|-----------|-------|
| Departure target | **2026-09-28** |
| Trip type | Expedition / trekking (not generic tourism) |
| Post-arrival logistics | Domestic travel toward eastern Nepal (Taplejung / Kangchenjunga access) |

**Route reliability matters more than raw cheapest price.**

### Trip characteristics

- Expedition / trekking travel
- Potentially large baggage (expedition duffels, trekking gear)
- Baggage loss is costly
- Missed connections are costly
- Extra visa requirements for transit are undesirable
- Self-transfer is allowed but must be penalized appropriately

### What this tool does NOT do

The goal is **NOT** to search all possible global flights.

The goal is to **evaluate only predefined route branches** described in configuration.

| Bad | Good |
|-----|------|
| Search all possible routes MOW → KTM | Evaluate branch `MOW → DXB → KTM` |
| Arbitrary multi-stop combinations | Only branches explicitly listed in YAML config |

### Example route branches

- Moscow → Doha → Kathmandu
- Moscow → Dubai → Kathmandu
- Moscow → Delhi → Kathmandu
- Moscow → Istanbul → Kathmandu
- Moscow → Chengdu → Kathmandu
- Moscow → Dubai (Aeroflot) + Dubai → Kathmandu (another airline) — self-transfer / mixed carrier

Some branches use a **single airline ticket**. Some use **mixed carriers**. Some use **self-transfer**. The system must compare them.

---

## Primary Goal

Build a **config-driven flight route evaluator**.

| | |
|---|---|
| **Input** | Route configuration (YAML) |
| **Output** | Bilingual HTML report (RU/EN), CSV export, JSON data; ranked routes with price, duration, connection, baggage, visa risk, self-transfer risk, score |

**No ticket purchasing is required.**

---

## Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | **Go** | Strong CLI tooling, YAML-friendly, concurrency for API calls, easy static binary, suitable for caching + reporting |

---

## Non-goals (Out of Scope for MVP)

- Ticket booking
- Payment
- OTA redirects
- User authentication
- Browser automation
- Scraping airline websites
- Personal data collection
- **Any database server** (PostgreSQL, MySQL, etc.)
- **SQL for flight queries** — all provider calls are HTTP/JSON
- **Provider historical price APIs** — price history is locally accumulated (Stage 4 JSONL), not fetched from providers

---

## Core Constraints

### 1. Route Search Scope

The application must **NOT** search arbitrary flight combinations.

It must **only** evaluate branches explicitly defined in config.

### 2. One-stop Preference

| In scope | Out of scope |
|----------|--------------|
| One-stop routes | Two-transfer routes |
| Self-transfer routes with one transfer (optional, config-controlled) | Arbitrary N-stop search |

Primary search scope: **one-stop routes**, optionally self-transfer with one transfer.

### 3. Visa Constraints

Extra transfer visas are undesirable.

| Category | Policy |
|----------|--------|
| **Preferred** | Airside transit without visa; visa-free transit; TWOV (Transit Without Visa) |
| **Penalized** | Uncertain transit rules; country-specific risk |
| **Rejected** | Transfer requiring explicit visa issuance |

Examples:

- Airside transit in Doha → good
- Self-transfer requiring immigration in some countries → risk
- Mandatory transit visa → reject

Config must allow **per-branch visa policy**.

### 4. Baggage Constraints

This trip may involve large expedition baggage.

Track and score:

- Checked baggage included?
- Baggage allowance (kg)
- Baggage cost if extra
- Sports baggage / oversize info if available from provider

| Condition | Action |
|-----------|--------|
| No checked baggage when required | Strong penalty |
| Unknown baggage info | Medium penalty |

### 5. Airline Coverage Constraints

Different APIs support different airlines. Providers may have partial coverage. The system works by combining providers.

The system must support **provider capability metadata** (see `ProviderCapabilities`).

**SU capability check (not a global requirement):**

- Do **not** require every run or config to include Aeroflot (SU).
- Do **not** require the registry to always contain an SU-capable provider.
- If config contains any leg with `preferred_airlines` including `SU`, the run must **warn** when no enabled provider declares SU coverage, and mark affected branches **unavailable**.

Example:

- Provider A declares SU coverage
- Provider B does not
- Branch leg with `preferred_airlines: [SU]` routes to Provider A when available; otherwise branch is unavailable with a clear reason

---

## Provider Model

Multiple providers supported. Examples:

- Aviasales / Travelpayouts
- Kiwi.com
- Amadeus

### Provider selection (MVP decision)

Both modes are supported:

1. **Explicit** — `provider_hint` on a leg or branch takes precedence.
2. **Automatic** — when no hint is set, select among enabled providers using capability metadata (see below). Prefer providers that declare coverage for the leg's `preferred_airlines`; if none declare it, fall back to providers with `AirlineCoverageMode: unknown` or `partial` (do not exclude them solely because an airline is missing from `SupportedAirlines`).

If no provider can serve a leg after search (including empty results), the branch is marked as **unavailable** (not a fatal error for other branches).

### Provider Interface

```go
type Provider interface {
    Name() string
    Capabilities() ProviderCapabilities
    Search(ctx context.Context, query Query) ([]Offer, error)
}

type ProviderCapabilities struct {
    SupportedAirlines       map[string]bool
    AirlineCoverageMode     AirlineCoverageMode
    SupportsSelfTransfer    bool
    SupportsBaggageInfo     bool
    SupportsRealTimePricing bool
}

type AirlineCoverageMode string

const (
    CoverageKnown    AirlineCoverageMode = "known"
    CoveragePartial  AirlineCoverageMode = "partial"
    CoverageUnknown  AirlineCoverageMode = "unknown"
)
```

**`SupportedAirlines` is advisory capability metadata**, not a complete airline universe unless the provider explicitly declares `AirlineCoverageMode: known` (`CoverageKnown`).

| Value | Meaning for auto-selection |
|-------|----------------------------|
| `known` (`CoverageKnown`) | `SupportedAirlines` is treated as the declared complete set |
| `partial` (`CoveragePartial`) | Map lists known-good airlines; provider may still return others |
| `unknown` (`CoverageUnknown`) | No reliable airline list; do not reject provider because an airline is absent from the map |

Do not exclude a provider from auto-selection solely because a `preferred_airline` is missing from `SupportedAirlines` when `AirlineCoverageMode` is `partial` or `unknown`.

Provider implementations live under `internal/provider/<name>/`. A mock provider is required for Stage 1.

---

## Date Search Policy (MVP decision)

| MVP | Post-MVP |
|-----|----------|
| **Exact departure date only** (`trip.departure_date`) | Flexible window (±N days) |

Rationale: expedition has a fixed departure target. Flexible date search is a future enhancement, not MVP.

---

## Data Model

### Route Branch

A branch represents a candidate route.

```yaml
branch:
  id: aeroflot_dubai
  name: Aeroflot via Dubai
```

Each branch contains **legs** and metadata (type, visa policy, connection time bounds).

### Branch types

| Type | Meaning |
|------|---------|
| `single_ticket` | One booking, one carrier or interline ticket |
| `mixed_carrier` | Multiple carriers, protected connection (single or combined ticket semantics per provider) |
| `self_transfer` | Separate tickets; passenger must collect baggage and re-check in |

### Leg

```yaml
from: MOW
to: DXB
preferred_airlines:
  - SU
provider_hint: aviasales   # optional; overrides auto-selection
```

### Offer

```go
type Offer struct {
    BranchID           string
    Provider           string
    Segments           []Segment
    Price              Money       // original currency from provider
    PriceNormalized    *Money      // scoring.currency equivalent for scoring; nil if conversion unavailable
    TotalDuration      time.Duration
    ConnectionDuration time.Duration
    CheckedBaggageKg   *int
    SelfTransfer       bool
    VisaRisk           Risk
}
```

### Money

```go
type Money struct {
    Amount   int64   // minor units (cents, kopecks) to avoid float drift
    Currency string  // ISO 4217, e.g. "USD", "RUB", "AED"
}
```

### Segment

All segment times must be stored as `time.Time` **with timezone** (`Location` set to the airport's local timezone).

```go
type Segment struct {
    From           string    // IATA code
    To             string    // IATA code
    Departure      time.Time // local time at origin airport (Location set)
    Arrival        time.Time // local time at destination airport (Location set)
    Airline        string    // IATA code, e.g. "SU"
    FlightNumber   string
    Duration       time.Duration
}
```

Resolve airport timezone from IATA code via a static lookup table (`internal/model/airport_tz.go` or similar). Do not assume UTC for display or scoring cutoffs.

### Branch evaluation result

```go
type BranchStatus string

const (
    StatusOK           BranchStatus = "ok"
    StatusUnavailable  BranchStatus = "unavailable"
    StatusRejected     BranchStatus = "rejected"
)

type BranchResult struct {
    BranchID    string
    Status      BranchStatus
    ReasonCodes []ReasonCode   // machine-readable; empty when StatusOK
    Offer       *Offer         // best offer when StatusOK
    Score       *float64
    // ...
}
```

### Reason codes (error taxonomy)

Unified machine-readable reason codes for unavailable/rejected branches. Used in `results.json`, CSV, and HTML (with bilingual human labels mapped from codes).

```go
type ReasonCode string

const (
    ReasonNoProvider           ReasonCode = "NO_PROVIDER"
    ReasonNoOffers             ReasonCode = "NO_OFFERS"
    ReasonConnectionTooShort   ReasonCode = "CONNECTION_TOO_SHORT"
    ReasonConnectionTooLong      ReasonCode = "CONNECTION_TOO_LONG"
    ReasonTransitVisaRequired  ReasonCode = "TRANSIT_VISA_REQUIRED"
    ReasonBaggageUnknown       ReasonCode = "BAGGAGE_UNKNOWN"
    ReasonAPIError             ReasonCode = "API_ERROR"
    ReasonCurrencyUnconvertible ReasonCode = "CURRENCY_UNCONVERTIBLE"
)
```

| Code | When |
|------|------|
| `NO_PROVIDER` | No enabled provider can serve a leg (incl. SU coverage gap) |
| `NO_OFFERS` | Provider returned zero matching offers |
| `CONNECTION_TOO_SHORT` | Layover below `min_connection_hours` |
| `CONNECTION_TOO_LONG` | Layover above `max_connection_hours` |
| `TRANSIT_VISA_REQUIRED` | Hub requires transit visa per static rules |
| `BAGGAGE_UNKNOWN` | Baggage info missing when `checked_required: true` (reject or strong penalty per branch policy; use code when rejecting) |
| `API_ERROR` | Provider HTTP/API failure after retries |
| `CURRENCY_UNCONVERTIBLE` | Offer currency cannot be converted to scoring currency |

Reports must show **both** the machine code and a bilingual human explanation. Multiple codes allowed per branch (ordered by severity).

### Risk levels

```go
type Risk string

const (
    RiskLow      Risk = "LOW"
    RiskMedium   Risk = "MEDIUM"
    RiskHigh     Risk = "HIGH"
    RiskRejected Risk = "REJECTED"
)
```

---

## Configuration Format

Full example config:

```yaml
trip:
  origin: MOW
  destination: KTM
  departure_date: 2026-09-28
  passengers: 1
  cabin: economy

constraints:
  max_stops: 1
  avoid_transfer_visas: true
  allow_self_transfer: true
  baggage:
    checked_required: true
    min_checked_kg: 23

cache:
  enabled: true
  ttl: 24h
  directory: .cache

history:
  enabled: true
  directory: history    # append-only JSONL price observations

providers:
  - id: aviasales
    enabled: true
  - id: kiwi
    enabled: true

scoring:
  currency: USD                 # normalize all prices to this currency for scoring
  weights:
    price: 1.0
    duration: 0.5
    baggage: 0.3
    visa: 1.0
    self_transfer: 0.8
    late_arrival: 0.4
  late_arrival_after: "18:00"   # local time at destination (KTM, Asia/Kathmandu)

branches:
  - id: qatar_doha
    name: Qatar via Doha
    type: single_ticket
    visa_policy: airside_only
    min_connection_hours: 2
    max_connection_hours: 8
    legs:
      - from: MOW
        to: DOH
        preferred_airlines: [QR]
      - from: DOH
        to: KTM
        preferred_airlines: [QR]

  - id: aeroflot_dubai_flydubai
    name: Aeroflot + Flydubai via Dubai
    type: self_transfer
    visa_policy: no_extra_visa
    min_connection_hours: 8
    max_connection_hours: 18
    legs:
      - from: MOW
        to: DXB
        preferred_airlines: [SU]
        provider_hint: aviasales
      - from: DXB
        to: KTM
        preferred_airlines: [FZ]
        provider_hint: kiwi

  - id: delhi_mixed
    name: Via Delhi
    type: mixed_carrier
    visa_policy: transit_only
    min_connection_hours: 5
    max_connection_hours: 12
    legs:
      - from: MOW
        to: DEL
        preferred_airlines: [SU, AI]
      - from: DEL
        to: KTM
        preferred_airlines: [AI, 6E]

  - id: istanbul
    name: Via Istanbul
    type: mixed_carrier
    visa_policy: airside_only
    min_connection_hours: 3
    max_connection_hours: 10
    legs:
      - from: MOW
        to: IST
        preferred_airlines: [SU, TK]
      - from: IST
        to: KTM
        preferred_airlines: [TK]

  - id: china_chengdu
    name: Via Chengdu
    type: mixed_carrier
    visa_policy: transit_or_twov
    min_connection_hours: 4
    max_connection_hours: 12
    legs:
      - from: MOW
        to: TFU
        preferred_airlines: [CA, 3U]
      - from: TFU
        to: KTM
        preferred_airlines: [CA, 3U]
```

### Visa policy values

| Value | Meaning |
|-------|---------|
| `airside_only` | Must stay airside; no immigration |
| `no_extra_visa` | Transit allowed without additional visa (includes TWOV where applicable) |
| `transit_only` | Standard transit rules; penalize uncertainty |
| `transit_or_twov` | Accept TWOV or visa-free transit; reject if visa required |

Config validation must reject unknown enum values and branches with invalid IATA codes, negative durations, or `min_connection_hours > max_connection_hours`.

---

## Currency Handling

Prices arrive from providers in **original currency**. Scoring and cross-branch comparison require a **single normalized currency**.

| Context | Rule |
|---------|------|
| **Reports (HTML, CSV, JSON)** | Show **original** price and currency from provider (e.g. `850 USD`, `72 000 RUB`) |
| **Scoring / ranking** | Normalize to `scoring.currency` (default **USD**) |
| **Conversion** | Static exchange-rate table for MVP (`internal/scoring/fx_rates.yaml` or embedded map); no live FX API required |
| **Missing conversion** | **Do not silently mix currencies.** If an offer's currency cannot be converted to `scoring.currency`, set `PriceNormalized = nil`, mark branch `unavailable` with reason `CURRENCY_UNCONVERTIBLE`, and log a warning |
| **Multi-leg branches** | Sum leg prices only after each leg is converted to scoring currency; if any leg fails conversion, branch is unavailable |

CSV may include both `price` + `currency` (original) and optional `price_usd` (normalized) columns.

---

## Timezones

Flight times are timezone-sensitive. Incorrect handling makes connection windows and arrival cutoffs ambiguous.

### Storage

- All `Segment.Departure` and `Segment.Arrival` values must be `time.Time` with `Location` set to the **local timezone of the departure/arrival airport**.
- Resolve timezone from IATA code via static airport → IANA timezone map (e.g. `MOW` → `Europe/Moscow`, `KTM` → `Asia/Kathmandu`, `DXB` → `Asia/Dubai`).

### Report display

- Show segment times in **local airport time** with timezone abbreviation or offset where helpful (e.g. `2026-09-28 14:30 MSK`, `2026-09-29 02:15 +0545`).
- Connection duration is computed from absolute instants (UTC internally), not by subtracting displayed local clock values across zones.

### Scoring cutoffs

- `scoring.late_arrival_after` applies to **final arrival at destination** (`trip.destination`, KTM) in **`Asia/Kathmandu` local time**.
- Example: arrival at KTM at 19:30 NPT with cutoff `18:00` → late arrival penalty applied.

### Tests

Include timezone fixtures (e.g. MOW→DXB→KTM crossing multiple zones) to verify connection math and late-arrival penalty.

---

## Scoring

Score combines multiple factors. **Lower score is better** (cost-minimization framing) OR define consistently — pick one and document in code. Recommended: **higher score is better** (0–100 scale) for readable reports.

### Factors

| Factor | Source |
|--------|--------|
| Price | Cheapest matching offer for branch |
| Total duration | Sum of flight + connection time |
| Layover duration | Connection time at hub |
| Baggage | Allowance vs `constraints.baggage` |
| Visa risk | Branch `visa_policy` + hub country rules (static lookup table for MVP) |
| Self-transfer risk | Branch `type: self_transfer` |
| Arrival time in Kathmandu | Penalize late arrivals per `scoring.late_arrival_after` in **Asia/Kathmandu** local time |

### Pseudo-formula

```
score =
    price_score
  + duration_score
  + baggage_score
  - self_transfer_penalty
  - visa_penalty
  - late_arrival_penalty
```

Normalize each component to a 0–100 sub-score before weighting. Final score is weighted sum using `scoring.weights`.

### Penalty rules

| Condition | Action |
|-----------|--------|
| Self-transfer with connection < `min_connection_hours` | Reject → `CONNECTION_TOO_SHORT` (+ self-transfer penalty if still scored) |
| Required transit visa | **Hard reject** → `TRANSIT_VISA_REQUIRED` |
| Connection below `min_connection_hours` | Reject → `CONNECTION_TOO_SHORT` |
| Connection above `max_connection_hours` | Reject → `CONNECTION_TOO_LONG` |
| No checked baggage when `checked_required: true` | Strong penalty |
| Unknown baggage info when rejection policy applies | Reject → `BAGGAGE_UNKNOWN`; otherwise medium penalty |

### Visa lookup (MVP)

Static map: hub IATA/country → transit visa requirement category. No external visa API for MVP. File: `internal/scoring/visa_rules.yaml` or embedded Go map.

---

## Caching

### MVP: filesystem cache only — no database

**No database is required for MVP.** Do not add PostgreSQL, MySQL, or any DB server. Do not use SQL for flight queries.

Provider API calls are plain **HTTP/JSON**, e.g.:

```
GET /search?from=MOW&to=DXB&date=2026-09-28
```

### Cache layout

```
.cache/
  aviasales/
    <hash>.json
  kiwi/
    <hash>.json
```

### Workflow

```
query → hash(key) → file exists and not expired?
                      yes → read JSON from disk
                      no  → call provider API → save raw JSON response
```

### Cache key

Hash of: `provider_id + from + to + date + passengers + cabin + leg-specific params`.

Use SHA-256 hex digest as filename.

### TTL

- Configurable via `cache.ttl` (Go duration string, e.g. `24h`, `6h`).
- On read: if `file_mod_time + ttl < now`, treat as miss and re-fetch.
- `cache.enabled: false` bypasses cache entirely.

### Requirements

- Store **raw JSON** responses (not normalized structs) for debuggability
- Avoid duplicate in-flight requests for the same key (mutex per key during fetch)
- `.cache/` must be in `.gitignore`

### Future (optional, not MVP)

SQLite single-file cache (`cache.db`) for TTL queries, dedup, and analytics. **Do not implement unless explicitly requested.**

---

## Output

### CLI

```bash
go run ./cmd/flight-routes \
  --config routes.yaml \
  --out ./out

# Mock / dry-run — no API keys required
go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out \
  --provider mock
```

| Flag | Required | Description |
|------|----------|-------------|
| `--config` | yes | Path to YAML config |
| `--out` | yes | Output directory for reports |
| `--provider` | no | Force provider mode: `mock` uses mock provider only (dry-run); default uses providers from config |
| `--verbose` | no | Log provider calls, cache hits/misses, reason codes |

### Mock / dry-run mode

`--provider mock` forces evaluation with the **mock provider only**, ignoring real API providers and API keys. Use for:

- Local development without credentials
- CI integration / golden tests
- Verifying report layout and ranking logic

When `--provider mock` is set, cache reads/writes for real providers are skipped.

### Outputs

| File | Role |
|------|------|
| `out/report.html` | **Primary** human-readable bilingual report (RU/EN) |
| `out/report.csv` | Machine-readable export |
| `out/results.json` | Normalized evaluation data for debugging / integration |
| `history/prices.jsonl` | Local accumulated price observations across runs (append-only; Stage 4) |

---

### HTML report (`out/report.html`) — primary format

**Static single-file report.** No server, no build step, no external CDN. Must open offline as a local file (`file://`).

#### Technical structure (MVP)

One self-contained HTML file with:

- **Embedded JSON** — normalized evaluation results (same schema as `out/results.json`)
- **Embedded translation dictionary** — RU and EN strings for all human-facing text
- **Inline CSS** — no external stylesheets
- **Small inline JS** — language switcher and expandable sections; no external scripts

Do not split into multiple HTML/CSS/JS files for MVP.

#### Bilingual requirements

| Requirement | Detail |
|-------------|--------|
| Languages | Russian (default), English |
| Switcher placement | **Top-right corner** of the report header |
| Switching behavior | Toggle RU / EN **without regenerating** the report (client-side only) |
| What changes on switch | All human-facing labels, headings, explanations, risk labels, recommendation text, table headers |
| What stays unchanged | Machine values: IATA codes, flight numbers, dates, prices, durations, scores, branch IDs |
| Language persistence (recommended) | Store selected locale in `localStorage` so reopening the report keeps the last chosen language |

Use `data-i18n` attributes or equivalent; store translations in an embedded JS object keyed by locale (`ru`, `en`).

#### HTML report sections (required)

1. **Header** — route title (e.g. MOW → KTM), trip date, generation timestamp
2. **Language switcher** — top-right: **RU / EN** toggle
3. **Summary ranking table** — all branches sorted by score (best first)
4. **Detailed expandable section per branch** — segments, connection time, provider(s), offer details
5. **Score breakdown** — per-component scores and penalties
6. **Risk badges** — visa, baggage, self-transfer (color-coded)
7. **Rejected / unavailable branches** — listed with human-readable reasons (bilingual)
8. **Final recommendation section** — top pick and brief rationale
9. **Offline-ready** — no external CDN or network dependencies
10. **Price dynamics** *(when history exists)* — per-branch price stats and trends (see Stage 4); bilingual table headers; hidden or omitted when no history file

#### Example header layout

```
┌─────────────────────────────────────────────────────────────┐
│  MOW → KTM — 2026-09-28              [ RU | EN ]  ← top-right│
│  Generated: 2026-07-05 11:30 UTC                             │
└─────────────────────────────────────────────────────────────┘
```

---

### JSON export (`out/results.json`)

Normalized structured output for debugging and downstream integration. Include:

- Trip metadata (origin, destination, date, passengers, cabin)
- Generation timestamp
- Per-branch evaluation results (status, offers, scores, score breakdown, rejection/unavailability reasons)
- Ranked order

This file is also embedded inside `report.html` (same data; may be a `<script type="application/json">` block or JS constant).

---

### CSV report (`out/report.csv`)

| Column | Description |
|--------|-------------|
| `branch_id` | Branch identifier |
| `branch_name` | Human-readable name |
| `status` | `ok`, `unavailable`, `rejected` |
| `reason_codes` | Comma-separated machine codes (empty when `ok`) |
| `price` | Numeric in **original** currency |
| `currency` | ISO 4217 (original) |
| `price_normalized` | Numeric in `scoring.currency` (empty if conversion failed) |
| `duration_minutes` | Total |
| `connection_minutes` | Layover |
| `baggage_kg` | Checked allowance or empty |
| `visa_risk` | LOW / MEDIUM / HIGH / REJECTED |
| `self_transfer` | true / false |
| `score` | Final weighted score |
| `provider` | Provider(s) used |

Sort CSV by score descending (best first). CSV is locale-neutral (English column headers, machine values only).

---

## Project Structure

```
cmd/
  flight-routes/
    main.go

README.md            # Setup, env vars, usage examples (required)

internal/
  config/          # YAML parse + validate
  model/           # Branch, Leg, Offer, Segment, Money, Risk
  provider/
    provider.go    # Provider interface + registry
    mock/          # Mock provider (Stage 1)
    aviasales/     # Stage 2
    kiwi/          # Stage 2
  search/          # Branch evaluation orchestration
  scoring/         # Score calculation, FX rates, visa rules, airport timezones
  report/          # HTML (bilingual), CSV, and JSON writers
  cache/           # Filesystem cache (hash, TTL, read/write)
  history/         # Append-only JSONL price observations (Stage 4)

configs/
  routes.yaml      # Example / default config for MOW→KTM expedition

out/               # Generated reports (gitignored)

.cache/            # Provider response cache (gitignored)

history/           # Price observation log (gitignored; append-only JSONL)

testdata/          # Golden files and test fixtures
  routes.yaml
  expected_report.html
  expected_report.csv
  expected_results.json
  config/          # Valid/invalid config snippets for validation tests
```

### Package responsibilities

| Package | Responsibility |
|---------|----------------|
| `config` | Load YAML, validate, expose typed struct |
| `model` | Domain types shared across packages |
| `provider` | Interface, registry, capability-based selection |
| `search` | For each branch: resolve legs → query providers → combine offers → filter by connection bounds |
| `scoring` | Apply weights and penalties; FX normalization; rank results |
| `report` | Write `report.html`, `report.csv`, and `results.json` |
| `cache` | Filesystem get/put with TTL; no SQL |
| `history` | Append-only JSONL price observations; read stats for report (Stage 4) |

---

## Testing Requirements

Automated tests are **required**, not optional. For this project, tests of deterministic business logic are **more important than real API integration** — the hardest part is route evaluation, scoring, and reporting, not HTTP calls.

All deterministic logic must be testable **without real provider APIs**.

### Test philosophy

| Type | Required | Notes |
|------|----------|-------|
| Unit tests | **Yes** | Config, provider selection, scoring, cache, report helpers |
| Integration tests | **Yes** | Mock provider + full pipeline (config → search → scoring → report) |
| Golden tests | **Yes** | Compare HTML / CSV / JSON output against committed fixtures |
| Real API tests | No | Optional / manual only; never in CI by default |
| Browser / UI tests | No | Not needed for MVP; no Playwright/Selenium |

Run tests with:

```bash
go test ./...
```

With coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**Minimum coverage:** 70% for packages under `internal/`. Focus on meaningful cases, not coverage for its own sake.

---

### Unit tests

#### Config validation (`internal/config`)

| Case | Expected |
|------|----------|
| Valid config | Parses successfully |
| Unknown `visa_policy` enum | Error |
| Invalid IATA code | Error |
| `min_connection_hours > max_connection_hours` | Error |
| Negative connection hours | Error |
| Missing required fields | Error |

Use table-driven tests with fixtures in `testdata/config/`.

#### Provider selection (`internal/provider`, `internal/search`)

| Case | Expected |
|------|----------|
| `provider_hint` set on leg | Hinted provider used, auto-selection skipped |
| No hint, provider declares airline coverage | Auto-select matching provider |
| Leg includes `SU` in `preferred_airlines`, no enabled provider serves SU | Branch marked unavailable; warning logged |
| Provider with `AirlineCoverageMode: partial` or `unknown` | Not auto-rejected solely because airline missing from map |

#### Scoring (`internal/scoring`)

Use deterministic fixtures (fixed offers, fixed weights). Example test names:

```go
func TestScore_CheaperRouteScoresBetter(t *testing.T)
func TestScore_VisaRejectRemovesBranch(t *testing.T)
func TestScore_PenalizesSelfTransfer(t *testing.T)
func TestScore_PenalizesMissingBaggage(t *testing.T)
func TestScore_PenalizesUnknownBaggage(t *testing.T)
```

| Case | Expected |
|------|----------|
| Lower price vs higher price (same branch type) | Lower price ranks higher |
| Mandatory transit visa | Branch rejected / excluded from ranking |
| `self_transfer` branch type | Self-transfer penalty applied |
| No checked baggage when required | Baggage penalty applied |
| Unknown baggage info | Medium baggage penalty applied |
| Currency not convertible | Branch unavailable → `CURRENCY_UNCONVERTIBLE` |
| Late arrival after cutoff (KTM local) | Late arrival penalty applied |

#### Reason codes (`internal/model`, report mappers)

Test that each `ReasonCode` maps to bilingual human text in HTML and appears in `results.json` / CSV.

| Case | Expected code |
|------|---------------|
| No provider for SU leg | `NO_PROVIDER` |
| Provider returns empty | `NO_OFFERS` |
| Layover too short | `CONNECTION_TOO_SHORT` |
| Unconvertible currency | `CURRENCY_UNCONVERTIBLE` |

#### Cache (`internal/cache`)

| Case | Expected |
|------|----------|
| Cache miss | API fetch path invoked; file written |
| Cache hit (valid TTL) | Cached JSON returned; no fetch |
| Expired entry (mtime + TTL < now) | Treated as miss; re-fetch |
| Corrupted JSON on disk | Treated as miss; re-fetch (do not crash) |

Use a temporary directory per test (`t.TempDir()`).

#### HTML report (`internal/report`)

No browser automation. Use **string / snapshot-style** assertions on generated HTML:

| Assertion | Example |
|-----------|---------|
| Report generated | Non-empty HTML output |
| RU labels present | Contains `Сводный рейтинг` or equivalent |
| EN labels present | Contains `Summary Ranking` or equivalent |
| Language switcher | Contains `RU` and `EN` toggle elements |
| Embedded data | Contains evaluation JSON payload |
| Offline-ready | No `http://` or `https://` CDN script/link tags |

---

### Integration tests

**Must-have.** End-to-end pipeline with mock provider only:

```
config → search → scoring → report
```

Setup:

- `testdata/routes.yaml` — sample config with known branches
- Mock provider returning **deterministic** offers per leg

Verify:

- Expected number of branches evaluated (e.g. 5)
- Expected rejections / unavailability (e.g. 1 rejected with known reason)
- Ranking order is **stable** across runs (same input → same order)
- All three outputs generated: `report.html`, `report.csv`, `results.json`

Place integration test in `internal/search/` or top-level `integration_test.go` under `cmd/flight-routes/`.

---

### Golden tests

Store expected outputs under `testdata/`:

```
testdata/
  expected_report.html
  expected_report.csv
  expected_results.json
```

After running the pipeline against `testdata/routes.yaml` + mock provider, compare actual output to golden files.

- Use `go test` with `-update` flag (or env var) to regenerate goldens when output format intentionally changes
- Document golden update workflow in test file comments
- Golden HTML comparison may normalize non-deterministic fields (e.g. generation timestamp) before compare

Golden tests are especially valuable for the report generator — they catch accidental regressions in bilingual labels, ranking table, and JSON schema.

#### History (`internal/history`) — Stage 4

| Case | Expected |
|------|----------|
| Append observation | New line appended to JSONL; file not truncated |
| Read stats | min/max/avg/previous computed per branch_id |
| Corrupt JSONL line | Skipped; remaining lines parsed |
| Empty history | Price dynamics section omitted from HTML |
| Trend calculation | Percent change vs previous and vs average |

---

## Development Stages

### Stage 1 — Skeleton (implement first)

- [ ] Go module init (`github.com/<user>/nepal` or similar)
- [ ] Config parser + validation
- [ ] Domain models
- [ ] Mock provider returning deterministic fake offers per leg
- [ ] Search orchestration (evaluate all branches)
- [ ] Basic scoring (price + duration + penalties)
- [ ] HTML (bilingual RU/EN) + CSV + JSON report generation
- [ ] CLI with `--config`, `--out`, and `--provider mock`
- [ ] `README.md` — setup, env vars, usage, opening report (see below)
- [ ] Example `configs/routes.yaml`
- [ ] `.gitignore` for `.cache/`, `out/`, secrets
- [ ] **Unit tests:** config validation, provider selection, scoring
- [ ] **Integration test:** mock provider + full pipeline (`testdata/routes.yaml`)
- [ ] **Golden tests:** `expected_report.html`, `expected_report.csv`, `expected_results.json`

**Stage 1 acceptance:**

1. `go run ./cmd/flight-routes --config configs/routes.yaml --out ./out --provider mock` produces ranked outputs using mock data (no API keys).
2. `go test ./...` passes with ≥70% coverage on `internal/` packages.

### Stage 2 — Real providers + cache

- [ ] Provider interface + registry
- [ ] Filesystem cache (`internal/cache`)
- [ ] Aviasales/Travelpayouts provider (env: API token)
- [ ] Kiwi provider stub or full implementation depending on available API access; if API access is unavailable, keep provider disabled and document setup in code/README
- [ ] Provider auto-selection by capabilities
- [ ] Respect `provider_hint` on legs
- [ ] Unit tests for cache (miss, hit, expired, corrupted JSON)

### Stage 3 — Scoring tuning

- [ ] Visa rules static table
- [ ] Baggage scoring refinement
- [ ] Self-transfer connection penalties
- [ ] Late arrival penalty
- [ ] Weight tuning via config

### Stage 4 — Price History / Tracking

**Goal:** Track price dynamics over time using **locally collected snapshots** from repeated runs.

**Important:** Most flight APIs do **not** provide reliable historical price curves. The app must **not** depend on provider historical price APIs. It builds its own local price history by appending observations from each run.

#### Storage

No database. Append-only **JSON Lines** file:

```
history/prices.jsonl
```

Each line is one observed snapshot (best offer per branch per run, or status-only when unavailable):

```json
{"observed_at":"2026-07-05T10:30:00Z","branch_id":"qatar_doha","provider":"aviasales","status":"ok","price":850,"currency":"USD","price_normalized":850,"duration_minutes":860,"score":91}
```

Unavailable/rejected branch (optional, no price):

```json
{"observed_at":"2026-07-05T10:30:00Z","branch_id":"delhi_mixed","status":"unavailable","reason_codes":["NO_OFFERS"]}
```

#### Requirements

- On every **successful run** (evaluation completes), **append** one observation per branch to `history/prices.jsonl`
- **Never overwrite** previous observations — append only
- If branch is unavailable/rejected, optionally append entry with `status` and `reason_codes` without price fields
- History directory configurable via config (`history.directory`); default: `history/`
- `history.enabled: false` skips append and hides Price dynamics section
- `history/` must be in `.gitignore`
- Use `price_normalized` (in `scoring.currency`) for cross-run min/max/avg/trend when available; fall back to original price only when all observations share the same `currency`

#### Report integration

When history file exists and has prior observations, HTML report includes **Price dynamics** section (bilingual).

Per branch show:

| Column | Description |
|--------|-------------|
| Route | Branch name |
| Current | Current run price (original currency displayed) |
| Previous | Last observed price for this branch |
| Min | Minimum observed (normalized) |
| Avg | Average observed (normalized) |
| Max | Maximum observed (normalized) |
| Trend | vs previous and vs average (e.g. `↑ +4.9%`) |
| Observations | Count of historical entries for branch |

Example table:

| Route | Current | Previous | Min | Avg | Max | Trend |
|-------|---------|----------|-----|-----|-----|-------|
| Qatar via Doha | $850 | $810 | $790 | $825 | $880 | ↑ +4.9% |

Trend formulas (use normalized prices):

- vs previous: `(current - previous) / previous × 100`
- vs average: `(current - avg) / avg × 100`

Show ↑ / ↓ / → with percentage; bilingual column headers via i18n.

#### Stage 4 checklist

- [ ] `internal/history` — append JSONL, read/parse, aggregate stats per branch
- [ ] Config `history.enabled` and `history.directory`
- [ ] CLI appends after successful evaluation
- [ ] HTML Price dynamics section (conditional on history)
- [ ] Unit tests — append, stats, corrupt line skip, empty history
- [ ] README note on price history accumulation

#### Future (optional, not Stage 4)

- Simple inline SVG price chart per branch
- Daily scheduled run (cron / launchd)
- Buy/wait recommendation based on trend

Do not implement unless SPEC is updated.

---

## README / Setup

The repository must include a **`README.md`** at the project root covering:

### Prerequisites

- Go 1.22+ (or current stable)
- No database setup required

### Quick start (mock / no API keys)

```bash
git clone <repo>
cd nepal
go test ./...
go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out \
  --provider mock
open ./out/report.html    # macOS; use xdg-open on Linux, start on Windows
```

### Production run (real providers)

```bash
export AVIASALES_TOKEN="..."
export KIWI_API_KEY="..."   # optional

go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out
```

### Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `AVIASALES_TOKEN` | For aviasales provider | Travelpayouts / Aviasales API token |
| `KIWI_API_KEY` | Optional | Kiwi.com API key (only if provider enabled and access available) |

### Opening the report

- Primary output: `out/report.html`
- Open directly in a browser as a local file (`file://`); no server needed
- Use the **RU / EN** switcher (top-right) to change language; choice persisted in `localStorage`

### Project docs

- Full specification: [`SPEC.md`](SPEC.md)
- Implementation plan: [`IMPLEMENTATION_PLAN.md`](IMPLEMENTATION_PLAN.md)
- Task board: [`TASKS.yaml`](TASKS.yaml)
- Reviewer instructions: [`REVIEWER.md`](REVIEWER.md)

---

## Environment / Secrets

Provider API keys via environment variables (never commit):

| Variable | Provider |
|----------|----------|
| `AVIASALES_TOKEN` | Aviasales / Travelpayouts |
| `KIWI_API_KEY` | Kiwi.com (optional; only if API access is available) |

If a provider is enabled in config but credentials are missing or API access is unavailable, log a warning and skip that provider (do not crash entire run).

---

## Acceptance Criteria (MVP complete = Stage 1 + Stage 2)

Command:

```bash
go run ./cmd/flight-routes \
  --config configs/routes.yaml \
  --out ./out
```

Must:

1. Parse and validate YAML config; exit non-zero on invalid config
2. Evaluate **only** branches defined in config (no global search)
3. Support `single_ticket`, `mixed_carrier`, and `self_transfer` branch types
4. Select providers by hint or capabilities; warn and mark branches unavailable when a leg includes `SU` in `preferred_airlines` and no enabled provider can serve SU for that leg (do not require SU globally)
5. Cache raw JSON responses under `.cache/<provider>/<hash>.json` with configurable TTL
6. Generate `out/report.html` (bilingual static HTML, offline-ready), `out/report.csv`, and `out/results.json`
7. Rank branches by score; show unavailable/rejected branches with **`ReasonCode`(s)** and bilingual explanations
8. Use exact departure date from config (no ±N days window)
9. Display prices in original currency; score using normalized `scoring.currency`; never silently mix currencies
10. Store and display segment times in airport-local timezones; late-arrival cutoff in **Asia/Kathmandu**
11. No database, no SQL, no booking flow
12. `go test ./...` passes; `internal/` packages meet ≥70% test coverage; integration + golden tests included
13. `--provider mock` works without API keys; documented in README

---

## Acceptance Criteria (Stage 4 — Price History)

After Stage 3 (or in parallel once Stages 1–2 are stable):

1. Each successful run appends one JSONL record per branch to `history/prices.jsonl` (never overwrites)
2. History directory configurable; default `history/`; directory gitignored
3. HTML report shows **Price dynamics** section when prior history exists
4. Per-branch stats: current, previous, min, avg, max, observation count, trend vs previous and vs average
5. No dependency on provider historical price APIs — history is locally accumulated only
6. Unit tests for append, aggregation, corrupt-line handling
7. `history.enabled: false` disables append and hides section

---

## Explicit Do-Not-Implement List

Cursor must **not** add unless this spec is updated:

- PostgreSQL / MySQL / any DB server
- SQLite cache (optional future only)
- **Provider historical price APIs** — do not fetch price history from Aviasales/Kiwi/etc.; local JSONL only
- Web UI or HTTP server (static `report.html` is required and is not a web app)
- User accounts
- Payment or booking
- Browser scraping
- Browser / UI automation tests (Playwright, Selenium, etc.)
- Arbitrary route search / "find all flights MOW-KTM"
- Two-transfer routes
- Flexible date windows (±N days)
- Inline SVG price charts, scheduled cron runs, buy/wait recommendations (Stage 4 future optional only)

**Allowed without DB:** append-only JSONL price history at `history/prices.jsonl` per Stage 4.

---

## Glossary

| Term | Definition |
|------|------------|
| **Branch** | A predefined route candidate (e.g. MOW→DOH→KTM) |
| **Leg** | One flight search unit within a branch (MOW→DOH) |
| **Offer** | A concrete priced itinerary returned by a provider |
| **Self-transfer** | Separate tickets; passenger responsible for connection |
| **TWOV** | Transit Without Visa |
| **Provider hint** | Config override forcing a specific provider for a leg |
| **Reason code** | Machine-readable branch failure code (e.g. `NO_OFFERS`, `CONNECTION_TOO_SHORT`) |
| **Price observation** | One append-only JSONL record capturing best-offer snapshot from a single run |

---

## Open Questions

Documented ambiguities resolved by default for autonomous implementation. Ask a human only if a chosen default proves wrong in practice.

### 1. Score direction

**Contradiction:** Scoring section mentions both "lower score is better" and recommends "higher score is better (0–100)".

**Default for implementation:** **Higher score is better** on a 0–100 scale. Penalties subtract from component sub-scores before weighting.

### 2. `BAGGAGE_UNKNOWN` — reject vs penalty

**Contradiction:** Baggage constraints say "unknown baggage → medium penalty"; penalty rules say "reject → `BAGGAGE_UNKNOWN`; otherwise medium penalty" without defining branch policy.

**Default for implementation:**

- If `constraints.baggage.checked_required: true` and baggage is unknown → **reject** with `BAGGAGE_UNKNOWN`
- Otherwise → **medium penalty**, branch remains rankable

### 3. CSV normalized price column name

**Contradiction:** Currency section mentions optional `price_usd` column; CSV section uses `price_normalized`.

**Default for implementation:** Use **`price_normalized`** (aligned with `scoring.currency`, not hardcoded USD).

### 4. Multi-leg offer assembly

**Gap:** Spec does not define how to combine per-leg search results for `single_ticket`, `mixed_carrier`, and `self_transfer` branches.

**Default for implementation:**

- Search each leg independently via selected provider(s)
- Combine leg offers where connection duration ∈ `[min_connection_hours, max_connection_hours]`
- For `self_transfer`, set `SelfTransfer: true` on combined offer
- Pick cheapest valid combination per branch after FX normalization; if none valid, reject with appropriate `ReasonCode`

### 5. Amadeus provider

**Gap:** Listed as example provider but not in Stage 2 deliverables.

**Default for implementation:** **Out of MVP.** Do not implement unless SPEC is updated.

### 6. Stage 1 vs acceptance timezone/currency requirements

**Contradiction:** Acceptance criteria #9–10 require currency normalization and timezones; Stage 3 lists "late arrival penalty" as tuning.

**Default for implementation:** Stage 1 must implement **basic** currency normalization, airport timezones, connection math, and late-arrival penalty. Stage 3 refines weights, visa rules depth, and tuning — not greenfield features.

### 7. Price history — same run vs prior runs

**Gap:** Stage 4 appends current run before or after displaying stats?

**Default for implementation:** Append **after** report generation. Price dynamics section compares current run results against history **excluding** the just-appended lines (previous = last line before this run).
