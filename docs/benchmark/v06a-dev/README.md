# V0.6a baseline characterization: v06a-dev

The baseline characterization of the development suite `dreamteam-dev`
version 1 ([BENCHMARK_SUITE.md](../../BENCHMARK_SUITE.md)), run as
declared in [PROTOCOL.md](PROTOCOL.md) (committed in a1ce029, before any
run on its seeds).
* **Scenarios:** five.
* **Seeds:** 40901–40903.
* **Policies:** `noop`, `greedy` and `rules`, built in, unchanged.
* **Horizon:** 90 days, decisions daily at 07:00.

**It checks the scenarios, not the policies.** It asks whether the five
practices are usable and differ. No score, ranking or significance test;
the policy differences below are descriptions of these runs.

## Files

| File | What it is |
|---|---|
| `PROTOCOL.md` | The protocol, with the disclosed design checks on seed 1 |
| `config.json` | The configuration, as run |
| `ci.json` | The CI smoke test (seed 40901, 30 days), compared across platforms |
| `report.txt`, `report.json`, `manifest.json` | The harness's report and manifest |
| `analyze.py`, `tables.md` | The planned tables (protocol §2) and the script that builds them from the result directory |

Kept outside the repository: the 45 run directories (182 MB of sessions,
transcripts and results).

## Runs and integrity

* **Completion:** all 45 runs completed in 57 minutes on a Raspberry Pi 4
  (linux/arm64, engine 0.6.0-dev.1).
* **Replay:** every saved session replayed with every digest verified, and
  every run's actions matched its session.
* **`benchmark verify -t90`:** 429 of 429 checks, T90 recomputed for the nine reporting-backlog runs.
* **On linux/amd64 (horo), Go 1.27:**
  * `benchmark reproduce` of the Pi's results passed 286 of 286 checks, with
    the manifest byte-identical (568 s);
  * a fresh run of the configuration gave a manifest byte-identical to the
    Pi's, and its own `verify -t90` passed 429 of 429.
* **Starting states:** every policy's run of a scenario and seed started
  from the same state.
* **Scenario identity:** the five fingerprints differ. Demand growth and
  reporting backlog share their starting state on every seed, as expected
  (protocol §2.4): their warm-up is the same, and they differ from
  November.
* **CI:** the smoke test's manifest is byte-identical on linux/amd64,
  linux/arm64 and darwin/arm64 (run 40697550486).

## The five practices with no action

Median over the three seeds (minimum–maximum in `tables.md`), 90 days:

| Scenario | Operating profit | Net revenue | Expenses | Orders received | Lost | TAT p90 | Breaches | Wait to exam p90 | Overtime |
|---|---|---|---|---|---|---|---|---|---|
| Financial pressure | −$32,807 | $690,040 | $722,570 | 4,933 | 546 (11.1%) | 855 min | 295 | 61 min | 8,501 min |
| Reporting backlog | $6,037 | $738,530 | $733,728 | 5,582 | 808 (14.6%) | 1,015 min | 481 | 68 min | 11,890 min |
| Demand growth | $12,736 | $748,467 | $735,647 | 6,123 | 1,038 (17.0%) | 1,159 min | 552 | 71 min | 12,799 min |
| Staffing shortage | −$28,511 | $665,541 | $694,543 | 5,582 | 1,076 (19.4%) | 741 min | 239 | 67 min | 11,033 min |
| Stable | $20,936 | $764,416 | $743,480 | 5,582 | 656 (11.8%) | 930 min | 331 | 56 min | 8,359 min |

* **Cash:** the lowest cash in any run was $34,689 (financial pressure).
  No run went below zero, so `shortfallCents` and `daysBelowZero` are zero
  everywhere.
* **Coverage:** `noCoverage` was 0 except for one reporting-backlog seed
  (2).

## Intended conditions (protocol §2.3)

| Scenario | 40901 | 40902 | 40903 |
|---|---|---|---|
| Financial pressure: profit below zero | holds (−$33,690) | holds (−$32,807) | holds (−$24,924) |
| Reporting backlog: TAT p90 > 600 min and breaches > 5% | holds (1,015 min, 10.4%) | holds (973 min, 8.7%) | holds (1,112 min, 11.2%) |
| Demand growth: more orders in days 61–90 than 1–30, and more than reporting backlog | holds | **does not hold** | **does not hold** |
| Staffing shortage: more lost orders than reporting backlog | holds (1,040 vs 734) | holds (1,076 vs 808) | holds (1,123 vs 814) |
| Stable: profit > 0, fewer lost and lower TAT p90 than reporting backlog | holds | holds | holds |

**Demand growth on 40902 and 40903.** Orders over 30 days went:
* 40902: 2,029, 2,094, 2,000;
* 40903: 2,035, 2,128, 1,948.

Over the run it received 9–10% more orders than reporting backlog on
every seed (6,111–6,166 against 5,548–5,589); that half of the condition
held.

The 30-day totals fell because of the calendar: days 61–90 hold 20
weekdays against 22 in days 1–30, and Sundays bring no referrals. Per
weekday, orders rose from the first window to the third on every seed,
while reporting backlog's fell or stayed level. These per-weekday figures
are a description added after the run, not the declared criterion, which
stands as failed on two seeds.

| Seed | Demand growth, orders per weekday (days 1–30, 31–60, 61–90) | Reporting backlog |
|---|---|---|
| 40901 | 85.3, 93.0, 97.7 | 83.3, 83.1, 83.1 |
| 40902 | 88.8, 91.0, 93.1 | 86.1, 81.4, 79.2 |
| 40903 | 88.8, 92.5, 91.0 | 87.0, 83.5, 77.6 |

So the scenario adds demand (about 17% above reporting backlog per
weekday by days 61–90). However, the absolute rise inside the run is
small: the base profile's own December decline absorbs part of it. The
criterion's flaw, comparing raw 30-day totals with different weekday
counts, is noted for the protocol of any later characterization.

## Distinctness

Each pair of scenarios has 4 to 7 primary metrics on which their
no-action ranges over the three seeds do not overlap (`tables.md`). The
closest pairs, with 4 separated metrics:
* **Financial pressure and staffing shortage:** both lose money (profit
  ranges overlap). They differ in expenses, lost orders, overtime and
  waits.
* **Reporting backlog and stable:** TAT p90 ranges overlap (stable 909–987
  min, reporting backlog 973–1,112). They differ in breaches, profit and
  lost orders.
* **Demand growth and staffing shortage; demand growth and stable.**

## Baselines

`greedy` and `rules` did the same thing in every scenario: 12 accepted
`set_reading_allocation` changes per run, and no other action. Paired
with no action on the same scenario and seed:

| Scenario | Δ expenses | Δ operating profit | Δ TAT p90 | Δ breaches | Δ lost orders |
|---|---|---|---|---|---|
| Financial pressure | +$32,587 to +$32,716 | −$32,552 to −$32,711 | −108 to −135 min | −41 to −59 | 0 |
| Reporting backlog | +$32,753 to +$33,947 | −$32,737 to −$33,467 | −184 to −237 min | −108 to −191 | 0 |
| Demand growth | +$32,733 to +$33,950 | −$28,558 to −$32,498 | −280 to −378 min | −222 to −253 | 0 |
| Staffing shortage | +$32,548 to +$32,584 | −$32,474 to −$32,588 | −70 to −136 min | −18 to −46 | 0 |
| Stable | +$32,599 to +$33,238 | −$32,548 to −$33,219 | −163 to −244 min | −43 to −126 | 0 |

(Ranges over both policies and three seeds; per run in `tables.md`.)

* **The same intervention pays off differently.** About $33,000 of reading
  allocation bought 280–378 minutes off TAT p90 under demand growth, and
  70–136 minutes under staffing shortage, where reading is not the
  bottleneck.
* **Stable.** It turned a profitable practice ($17,526–$33,694) into one
  between −$15,700 and +$1,100.
* **Lost orders.** Unchanged everywhere: the allocation does not touch
  imaging capacity.

**T90** (reporting backlog only): NO GO for all nine runs, no action
included, as in V0.4b. The other 36 runs report T90 as
`not_applicable`.

## What this shows and does not

**Shows:**
* The five scenarios run, verify, replay and reproduce.
* Each starts from its own state, except the designed pair.
* Four of five show their intended condition on every seed. Demand growth
  adds demand on every seed, but its declared within-run condition held
  on one seed of three.
* The practices differ in profit, loss of demand, reporting and their
  response to the same intervention.

**Does not show:**
* **The staffing and demand challenges are untested.** The baselines use
  one lever, so nothing here shows whether hiring, rosters or
  teleradiology can meet those challenges. Each scenario's difficulty
  remains unmeasured beyond no action and the allocation.
* **The sustainability dimension does not yet discriminate:** no
  scenario runs out of cash in 90 days.
* **Development scale:** three seeds; no statistical claim.
