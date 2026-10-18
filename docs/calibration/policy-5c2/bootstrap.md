# Cluster-aware pass estimate (exploratory)

Slice 5C.2 diagnostic: `block_bootstrap.py` on the `p5-eval3` records
(development seeds 901–910, engine 0.2.0-dev.13), 1,000 replicates of a
40-seed evaluation.

* Each replicate draws 40 seeds with replacement from the plan's ten.
* Within each seed it resamples whole Monday-based weeks, with
  replacement, inside T90 and inside the tail. The run's day and week
  dependence is kept.
* It then applies the frozen per-seed gates, the joint rule (at most 4
  of 40 failing) and the pooled rates.

**Exploratory.** It is no gate and selects nothing, and two limits
bound what it can show:

* between-seed variation is only what ten seeds contain;
* resampling weeks independently drops week-to-week carry-over.

| Under independence (5C.1 estimate) | Per-seed failure | 40 seeds: P(more than 4 fail) |
|---|---|---|
| D2-full-reading-alloc65 | 0.059 | 0.086 |
| E2, E3 | 0.000 | 0.000 |

Week-block bootstrap:

| Plan | Replicates | P(pass joint rule, 40 seeds) | P(pass joint rule and pooled rates) | Mean failing seeds | P(seed fails) |
|---|---|---|---|---|---|
| A-no-action | 1000 | 0.000 | 0.000 | 39.96 | 0.999 |
| D-full-reading | 1000 | 0.000 | 0.000 | 27.98 | 0.699 |
| E-urgent-tele | 1000 | 0.000 | 0.000 | 17.04 | 0.426 |
| D2-full-reading-alloc65 | 1000 | 0.334 | 0.334 | 5.57 | 0.139 |
| E2-urgent-tele-alloc65 | 1000 | 1.000 | 1.000 | 0.33 | 0.008 |
| E3-urgent-weeknights-alloc65 | 1000 | 1.000 | 1.000 | 0.33 | 0.008 |
| F-broad-tele | 1000 | 1.000 | 1.000 | 0.00 | 0.000 |
| H-weeknight-tele | 1000 | 1.000 | 1.000 | 0.07 | 0.002 |

D2's per-seed failure is 14% under the run's own dependence, against
5.9% under independence, almost all of it urgent. So D2 would pass the
40-seed joint rule about a third of the time. E2 and E3 fail a seed
0.8% of the time and pass every replicate. This is the cluster-aware
evidence on the D2/E3 choice that the independence model could not give.
It is for the owner's decision and does not decide it.
