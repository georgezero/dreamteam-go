# Repeated refused proposals, version 2

`dreamteam.repeat-evidence/v2`, rule `REPEATED_REFUSED_PROPOSAL` version 2, beside the frozen v1 rule. A repeat is the identical action (canonical JSON without requestId) proposed again after a refusal; the refusal's wording plays no part.

| Group | Model | Refusals | insufficient_funds | repeated_refused | repeated_accepted | repeated_unvalidated | not_repeated | unresolved | insufficient_evidence | with a related (modified) proposal | findings |
|---|---|---|---|---|---|---|---|---|---|---|---|
| probes-v2-format | scripted probe | 7 | 7 | 1 | 0 | 0 | 0 | 3 | 3 | 0 | 1 |
| probes | scripted probe | 7 | 7 | 2 | 0 | 0 | 5 | 0 | 0 | 2 | 1 |
| v05a | Qwen3.6-35B-A3B-oQ4e-mtp | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 0 |
| v05b | Qwen3.6-35B-A3B-oQ4e-mtp | 8 | 0 | 0 | 0 | 0 | 0 | 1 | 7 | 0 | 0 |
| v05c | Qwen3.6-35B-A3B-oQ4e-mtp | 103 | 0 | 0 | 0 | 0 | 5 | 14 | 84 | 6 | 0 |
| v05c | gemma-4-26B-A4B-it-oQ4e-mtp | 22 | 0 | 0 | 0 | 0 | 0 | 8 | 14 | 8 | 0 |
| v05c | gemma-4-31B-it-oQ4e-mtp | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 0 |
| v06c | Qwen3.6-35B-A3B-oQ4e-mtp | 186 | 0 | 0 | 0 | 0 | 7 | 20 | 159 | 7 | 0 |
| v06c | gemma-4-26B-A4B-it-oQ4e-mtp | 32 | 0 | 0 | 0 | 0 | 0 | 9 | 23 | 6 | 0 |
| v07c | Qwen3.6-35B-A3B-oQ4e-mtp | 137 | 0 | 0 | 0 | 0 | 2 | 28 | 107 | 11 | 0 |
| v07c | gemma-4-26B-A4B-it-oQ4e-mtp | 68 | 0 | 0 | 0 | 0 | 0 | 13 | 55 | 9 | 0 |

## Versus v1 (repeat links: one refused occurrence to the next of the same proposal)

| Group | Model | v1 links | v2 links | both | v1 only: v2 abstains | v1 only: different actions | v2 only: message differs | v2 only: other |
|---|---|---|---|---|---|---|---|---|
| v05a | Qwen3.6-35B-A3B-oQ4e-mtp | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| v05b | Qwen3.6-35B-A3B-oQ4e-mtp | 3 | 0 | 0 | 3 | 0 | 0 | 0 |
| v05c | Qwen3.6-35B-A3B-oQ4e-mtp | 42 | 0 | 0 | 41 | 1 | 0 | 0 |
| v05c | gemma-4-26B-A4B-it-oQ4e-mtp | 8 | 0 | 0 | 6 | 2 | 0 | 0 |
| v05c | gemma-4-31B-it-oQ4e-mtp | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| v06c | Qwen3.6-35B-A3B-oQ4e-mtp | 56 | 0 | 0 | 55 | 1 | 0 | 0 |
| v06c | gemma-4-26B-A4B-it-oQ4e-mtp | 12 | 0 | 0 | 10 | 2 | 0 | 0 |
| v07c | Qwen3.6-35B-A3B-oQ4e-mtp | 44 | 0 | 0 | 42 | 2 | 0 | 0 |
| v07c | gemma-4-26B-A4B-it-oQ4e-mtp | 37 | 0 | 0 | 37 | 0 | 0 | 0 |
