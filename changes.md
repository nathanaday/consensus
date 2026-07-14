# Changes — Phase 3 analysis tools

Date: 2026-07-13

This document records a gap analysis of the Consensus tool set against the
questions a real IoT user asks in natural language, the tools added to close
those gaps, and the items considered but deliberately deferred. All changes
follow the existing conventions: one tool per file, registration only in
`register.go`, pure math in `internal/analysis`, compact statistics out —
never bulk rows.

## Method

The tool set was reviewed by role-playing a user with many sensor channels
(temperature, vibration, sound, power, humidity) asking questions of an LLM
connected to this server. Each question was mapped to the tool chain that
answers it; questions with no viable chain became gaps.

Questions already answered well: "what data do I have" (list_datasets,
describe_dataset), "summarize this channel" (summary_stats), "anything
unusual?" (detect_outliers, compare_to_baseline), "is there a trend?"
(fit_trend), "is this sensor healthy?" (data_quality), "do these move
together?" (correlate), "this week vs last week?" (compare_datasets), "shape
over time?" (profile), plus the transforms (copy, remove_outliers, resample).

Questions with no viable chain, and the tool added for each:

## New tools

### 1. `find_events` — "When was the temperature above 80, and for how long?"

The single most common IoT question is a threshold question, and nothing
answered it. `detect_outliers` finds statistical outliers, but a user's
threshold is a domain fact (a safety limit, a spec ceiling), not a quartile.
The tool takes a condition — `above`/`below` a threshold, or
`between`/`outside` a band — and groups matching points into events with
start, end, duration, and peak, reusing the same episode-grouping tolerance
`compare_to_baseline` uses so "an event" means the same thing everywhere.
It also reports total time and percent of the window in the condition, which
directly answers "how long" and "how often".

### 2. `overview` — "Give me the lay of the land across all my sensors."

Every analysis tool was per-dataset, so a fleet question ("what's the current
state of all my channels?", "which sensor looks off?") forced one call per
dataset — slow and token-hungry with dozens of channels. `overview` returns
one compact block per dataset (id, unit, origin, row count, time range, last
reading, mean/min/max/stddev) in a single call, with an optional id prefix
filter. It is the natural first call of a session; the LLM then drills into
one channel with the single-dataset tools.

### 3. `distribution` — "What does the spread look like? What's the p95?"

`summary_stats` reports mean/median/stddev but nothing about shape: no
percentiles, no histogram, no way to see skew or bimodality (a vibration
channel with two modes is a classic IoT signature). The tool reports the
standard percentile ladder (p05–p99) and a bounded equal-width histogram
(auto bin count from sample size, cap 40).

### 4. `seasonal_profile` — "Is there a daily pattern? What does a typical day look like?"

Deferred from phase 2 as "seasonality detection", this is answered here with
a deliberately simple, explainable model rather than spectral analysis:
group values by cycle position (hour of day, or day of week) and report each
position's mean/min/max/count plus `cycle_strength` — the share of total
variance explained by the position (eta squared). Strength near 1 with 24
clean positions answers "yes, strong daily cycle"; near 0 answers "no". The
same output is the "typical day" narration for free. Positions use UTC, and
a caveat says so, since local-time patterns may appear shifted.

### 5. `find_lag` — "Does vibration follow temperature? By how much?"

`correlate` answers "do they move together" but deliberately deferred lag.
The infrastructure it built (shared bucket grid, aligned means, Pearson)
makes lag scanning a small, low-risk extension: shift one series by whole
buckets in both directions, report the shift with the strongest absolute
correlation alongside the zero-lag coefficient for comparison. Positive lag
means B follows A. Caveats flag weak best-lag correlation, estimates at the
scan edge (suggesting a wider `max_lag`), and thin alignment. Kept as its
own tool, not a `correlate` mode, per the design rule that single-purpose
tools beat mode-switched ones.

### 6. `integrate` — "How much energy did we use this week?"

`rate_of_change` differentiates, but nothing integrated — yet many IoT
channels are rates whose integral is the business quantity (power to energy,
flow to volume). Trapezoidal integration reports value-seconds and
value-hours (a kW channel reads directly as kWh), plus `time_weighted_mean`
— the honest average for unevenly sampled data, where the point mean
over-weights bursts of fast sampling. Long sampling gaps are integrated as
straight lines and flagged in a caveat rather than silently absorbed.

### 7. `value_at` — "What was the temperature at 3pm yesterday?"

Point-in-time lookup had no direct answer; the workaround (summary_stats
over a sliver window) fails when no sample lands in the window and is a
clumsy chain for such a simple question. The tool returns the nearest sample,
its signed offset from the requested instant, and a linearly interpolated
estimate when the instant is inside the data's span. An optional
`max_distance` turns "nothing nearby" into a hard error. Returns a single
point, which stays within the compact-output mission.

## Changes to existing code

- `internal/tools/correlate.go`: the dataset-pair loading and
  overlap-windowing logic was extracted into `overlappingRows` (in
  `find_lag.go`) and is now shared by `correlate` and `find_lag`, removing
  ~50 duplicated lines. Behavior is unchanged; existing correlate tests
  still pass.
- `internal/store/info.go`: `Capabilities()` still described only ingest and
  cataloging — stale since phase 1. It now truthfully summarizes lineage,
  single-series analysis, and cross-channel analysis, so `server_info`
  answers "what can you do?" accurately.
- `cmd/server/integration_test.go`: a phase-3 story test drives the new
  tools end to end over stdio against a fixture with a known daily cycle, a
  known 2-hour cross-channel lag, known threshold events, and a known
  integral.

## New analysis primitives

Each tool's math lives in `internal/analysis` as pure functions with table
tests, per the phase-2 pattern: `events.go` (condition matching and episode
grouping), `distribution.go` (percentiles and histogram), `seasonal.go`
(cycle positions and eta squared), `lag.go` (bucket-shift correlation scan),
`integrate.go` (trapezoidal integral), `at.go` (nearest and interpolated
point).

## Considered and deferred (not rejected)

- **Change-point detection** ("when did the behavior shift?"). Real value,
  but doing it honestly needs a defensible statistical model (binary
  segmentation with a penalty, or Bayesian online detection); a naive
  version would confidently report noise. `compare_to_baseline` plus
  `profile` covers the near-term need. Revisit with real usage.
- **Derived arithmetic channels** ("channel A minus channel B", unit
  conversion). A `derive` transform is powerful but opens an expression
  language design question; `compare_datasets` and `correlate` answer the
  common motivating cases. Worth a dedicated design pass if users ask for
  computed channels.
- **Rolling-average smoothing.** `resample` already smooths; a distinct
  rolling transform adds little until someone needs aligned-timestamp
  smoothing specifically.
- **Timezone-aware seasonal profiles.** `seasonal_profile` is UTC-only and
  says so in a caveat. A `timezone` input is a clean later addition once the
  ingest path records or accepts zone information.
- **Removing the `greet` scaffold tool.** It is noise in the tool list an
  LLM sees, but it is also the canonical example in CLAUDE.md and the
  scaffold tests; removal is a housekeeping decision left to the repo owner.
