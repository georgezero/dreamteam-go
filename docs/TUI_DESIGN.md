# DreamTeam — TUI Design

Status: **draft for review**.
Companion documents: [PLAN.md](PLAN.md), [ROADMAP.md](ROADMAP.md),
[SIMULATION_DESIGN.md](SIMULATION_DESIGN.md).

The TUI is the first way anyone experiences DreamTeam, and V0.1 must be
genuinely playable in it. It is built with **Bubble Tea v2** and **Lip Gloss
v2** (module paths `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`;
versions pinned in `go.mod` at implementation time). `charm.land/bubbles/v2`
components (viewport, table, help) may be used where they save real effort;
nothing else.

All values in mockups below are illustrative, not simulation output.

---

## 1. Principles

1. **The TUI is a client.** It holds UI state only (current screen, selection,
   scroll, speed, paused flag). It reads `api.Observation` from the session and
   sends `api.Action`s. It never imports `internal/sim`.
2. **Rendering never affects the simulation.** Frame rate, terminal size, speed
   and pauses change *when* you see things, not *what happens* (chunking
   invariance, SIMULATION_DESIGN §3.4).
3. **Flow is visible.** Patients should visibly move: arrivals in the log,
   queues growing, rooms flipping between states, the unread count rising and
   falling, cash ticking.
4. **Explain, don't just display.** Alerts say what's wrong *and* where to look
   ("US running 15m behind — 3 waiting · [s] schedule").
5. **Never color-only.** Every status has a glyph and/or text; respects
   `NO_COLOR`; works in 16-color terminals.
6. **Starts paused.** The player gets a moment to read the morning situation.

---

## 2. Run loop and pacing

```
            ┌───────────── tickMsg every 100 ms (10 fps) ─────────────┐
            ▼                                                         │
 Update: if running:                                                  │
           simDelta = speed × wallElapsed      (wall time measured     │
           (Max: run until event budget/frame time exhausted)  in TUI) │
           reason = session.Advance(simDelta, stopOnAlerts)           │
           if reason == AutoPause → paused, flash alert               │
         obs = session.Observe()  (cached until next advance/action) ─┘
 View:   render(obs, uiState, width, height)
```

* The simulation runs **synchronously inside `Update`**, not in a separate
  goroutine. Each frame advances a bounded amount. This avoids locks and data
  races and still feels "in the background" because frames are short.
  Tradeoff: at Max speed the UI frame is the unit of work, so we cap work per
  frame (e.g., 25 ms or 20k events), checked only **between timestamp
  transactions**, never inside one, to stay responsive. A goroutine-based engine
  would make Max a bit faster but adds synchronization and makes "pause exactly
  here" harder. Not worth it at ~1,500 events/day.
* Wall time is read only in the TUI layer. The engine never sees it.
* 10 fps is plenty for a dashboard and keeps CPU low on a Raspberry Pi.
  Configurable (`--fps`).

### Speeds

Speeds are expressed in **simulated time per real second** — clearer than "10×".

| Key | Speed | A 12-hour weekday takes |
|---|---|---|
| `1` | 1 min/s | 12 min |
| `2` | 5 min/s | 2.4 min |
| `3` | 15 min/s | 48 s |
| `4` | 1 h/s | 12 s |
| `5` | Max | as fast as possible (seconds for a month) |

Overnight (closed hours) is skipped automatically at any speed: the session jumps
to the next observable entry when the practice is closed and nothing is happening
on screen, so nobody watches an empty building for minutes.

### Step controls

| Key | Control | Semantics |
|---|---|---|
| `space` | run / pause | |
| `n` | next event | advance to the next **observable** log entry (never internal events — see SIMULATION_DESIGN §9.3) |
| `N` | +15 minutes | |
| `e` | end of day | run to the **operational day end** (practice closed, nobody on site, no exam in progress; SIMULATION_DESIGN §3.4.2), then the daily summary. Financials in it are "as of operational close"; the day's P&L is final at accounting close (23:59:59) |
| `E` | next morning | run until 06:00 next operating day |

### Auto-pause

Toggles in Settings (`,`). Defaults on: urgent order arrives, room goes down,
staff absence, urgent read overdue, end of day. Default off: every walk-in,
every no-show. Auto-pause only reacts to observable entries.

---

## 3. Layout system

Breakpoints by terminal size:

| Class | Size | Layout |
|---|---|---|
| Wide | ≥ 140 × 40 | three columns + full-width log; side detail pane |
| Standard | 100–139 × 30+ | three columns compressed; log 6 lines |
| Compact | 80–99 × 24+ | two columns; reduced labels; log 4 lines |
| Minimal | < 80 × 24 | single stacked column with section tabs (`[` `]`) |
| Too small | < 60 × 16 | "Terminal too small (need 60×16)" |

Every screen shares a **top bar** (practice, day/time, run state, speed, cash,
today's counts) and a **bottom bar** (context key hints). Screens switch with a
single key from anywhere; `esc` goes back; `enter` drills in; `?` help.

Global screen keys:

| Key | Screen |
|---|---|
| `d` | Dashboard |
| `s` | Schedule (modality timelines) |
| `p` | Patients (list → journey) |
| `t` | Staff |
| `w` | Worklist (radiology) |
| `f` | Finance |
| `l` | Event log |
| `a` | Actions menu |
| `?` | Help / field guide |
| `q` / `ctrl+c` | quit (confirm if unsaved — V0.2) |

Navigation: arrows or `h j k l`, `g`/`G` top/bottom, `/` filter in lists,
`tab` cycles panes.

Mouse (V0.2): click to select rows, wheel to scroll. Not required for V0.1.

### Visual vocabulary

| Glyph | Meaning |
|---|---|
| `●` | active / scanning / busy |
| `○` | idle / available |
| `◐` | on break |
| `✕` | down / absent / no-show |
| `▲` `▼` | change up / down |
| `⚠` | warning alert |
| `‼` | urgent |
| `─` | unstaffed / closed time on timelines |

Colors (semantic, via a small palette): green = healthy/on time, amber =
behind / at risk, red = breach/down, cyan = informational, dim = closed /
unstaffed. Money: green positive, red negative.

---

## 4. Screens

### 4.1 Dashboard (Wide, 140×40)

```
┌ DREAMTEAM · DreamTeam Imaging Center ─────────────────────────── Tue Oct 13 · Day 1 · 10:42 ─── ▶ RUNNING 15 min/s ─┐
│ Cash $142,850 ▼$1,940    Revenue today $11,940    Expenses today $8,615    Op. profit today $3,325    A/R $412,300  │
├─ ROOMS ──────────────────────────────────────────────┬─ PATIENT FLOW ────────────────┬─ READING ────────────────────┤
│ XR  ● Scanning  P01142 Chest 2V   6/10m   Util 54%   │ Booked today          71       │ Dr. Avery  ● Reading CT A/P │
│     Tech Riley (XR+CT)    Waiting 2   Next 10:45     │ Arrived               38       │ Dr. Blake  ○ starts 11:30   │
│ US  ● Scanning  P01151 Abd compl 22/45m  Util 92%    │   walk-ins             4       │                              │
│     Tech Morgan     ⚠ 15m behind  Waiting 3          │ Waiting               6  ⚠     │ Unread      12   ‼ 2 urgent │
│ CT  ○ Idle — tech shared with XR   Util 61%          │ In exam               2        │ Oldest      3h 12m          │
│     Tech Riley (shared)   Waiting 1   Next 10:45     │ Done (left)          30        │ TAT median  2h 05m          │
│ MR  ● Scanning  P01139 Brain w/o 18/45m  Util 82%    │ No-show / cancel   2 / 3       │ Urgent TAT  34m   ✓ ≤60m    │
│     Tech Casey      Waiting 0   Next 11:15           │ Remaining today      27        │ Final today 26              │
├─ STAFF ──────────────────────────────────────────────┴────────────────────────────────┴──────────────────────────────┤
│ Desk Jordan ● registering   Ops Taylor ○ admin   Techs on duty 3/4 (Sam 11:30)   Rads 1/2   OT today 0m   Breaks 0 skipped │
├─ ALERTS ─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ⚠ US running 15m behind — 3 waiting                                                                    [s] schedule │
│ ⚠ CT and XR share one technologist until 11:30 — 1 CT patient waiting                                  [t] staff    │
│ ⚠ 2 urgent studies unread — oldest 21m                                                                 [w] worklist │
├─ EVENT LOG ──────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ 10:42  Walk-in P01163 arrived for XR extremity · waiting                                                           │
│ 10:41  CT P01136 report finalized by Dr. Avery (TAT 1h 32m) · revenue $395                                          │
│ 10:39  US P01151 arrived 9m late for 10:15                                                                          │
│ 10:38  XR P01142 exam started (Riley)                                                                               │
│ 10:36  Payment received: claim C00811 (Commercial) $612                                                             │
│ 10:35  MR P01139 exam started (Casey)                                                                               │
├──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ space pause  1-5 speed  n next  e end of day   d dash  s sched  p patients  t staff  w worklist  f finance  a act  ? │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

Room `Util` on the dashboard is staffed-hours utilization so far today. The
end-of-day summary shows both definitions (SIMULATION_DESIGN §11).

### 4.2 Dashboard (Compact, 80×24)

```
DREAMTEAM  Tue D1 10:42  ▶15m/s   Cash $142,850 ▼   Profit today $3,325
─ ROOMS ─────────────────────────────── ─ FLOW ──────────────────────────
XR ● P01142 Chest   54%  w2            Booked 71  Arrived 38 (WI 4)
US ● P01151 Abd     92%  w3 ⚠+15m       Waiting 6 ⚠  In exam 2
CT ○ shared tech    61%  w1            Done 30   NS/Cx 2/3
MR ● P01139 Brain   82%  w0            Unread 12 ‼2  TAT 2h05
─ STAFF ─────────────────────────────────────────────────────────────────
Desk ● Jordan  Ops ○  Techs 3/4  Rads 1/2  OT 0m
─ ALERTS ────────────────────────────────────────────────────────────────
⚠ US 15m behind, 3 waiting           ⚠ 2 urgent unread (21m)
─ LOG ───────────────────────────────────────────────────────────────────
10:42 Walk-in P01163 → XR extremity
10:41 CT P01136 report final (Avery) +$395
10:39 US P01151 arrived 9m late
10:38 XR P01142 exam started
spc pause 1-5 speed n next e EOD  s p t w f l a ?
```

### 4.3 Schedule (modality timelines)

Each character is 15 minutes; 08:00–20:00 = 48 columns. The cursor selects a
room; the list below shows that room's appointments.

```
┌ SCHEDULE · Tue Oct 13 ──────────────────────────────────────────── 10:42 ── ▶ 15 min/s ┐
│        08   09   10   11   12   13   14   15   16   17   18   19   20                 │
│ XR   ██████████▓▓░░░░░░░░░░◐░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░    staffed 08–20 │
│ US   ────────█████▓▓▓░░░░░░░░░◐░░░░░░░░░░░░░░░░░░░────────────────    staffed 09:30–18│
│ CT   ███·█·██▓░░░░·░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░    shared until 11:30│
│ MR ▶ ████×███▓▓▓░░░░░░░░░░◐░░░░░░░░░░───────────────────────────    staffed 08–16:30 │
│                    ^ now                                                                │
│ █ done  ▓ in progress  ░ booked  · open slot  × no-show  ◐ tech break  ─ unstaffed      │
├─ MR · 9 booked · 1 no-show · 0 open · util so far 82% (staffed hrs)           ─────────┤
│  TIME   PATIENT  EXAM             PRI   STATUS           ARRIVED   WAIT                 │
│  08:00  P01102   MR knee w/o      R     ✓ done 08:48     07:51       4m                 │
│  08:45  P01107   MR L-spine w/o   R     ✕ no-show                                       │
│  09:30  P01118   MR brain w/wo C  R     ✓ done 10:31     09:12      11m                 │
│▶ 10:30  P01139   MR brain w/o     R     ● scanning 18m   10:21       2m                 │
│  11:15  P01147   MR shoulder w/o  R     booked                                          │
│  12:00  P01150   MR brain w/wo C  S     booked                                          │
│  …                                                                                      │
│  (no MR slots after 16:30 — no MR technologist on duty)                                 │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│ ↑↓ room  j/k appointments  enter patient  ←→ day  esc back                              │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

`←/→` pages days (future days show bookings only; past days show outcomes).

### 4.4 Patients — list and journey

List: filter by state (`waiting`, `in exam`, `unread`, `all`), sort by wait.

```
┌ PATIENTS · waiting (6) ──────────────────────────────────────────── 10:42 ───┐
│  PATIENT  MODALITY  EXAM              APPT   ARRIVED  WAIT  STATE     PRI   │
│▶ P01151   US        Abdomen complete  10:15  10:24    18m   waiting   R    │
│  P01153   US        Pelvis            10:30  10:11    31m ⚠ waiting   R    │
│  P01160   US        Thyroid           11:00  10:35     7m   waiting   R    │
│  P01163   XR        Extremity         walk   10:42     0m   registering R  │
│  P01148   CT        Head w/o          10:45  10:28    14m   waiting   S    │
│  P01155   XR        Chest 2V          10:45  10:30    12m   waiting   R    │
├──────────────────────────────────────────────────────────────────────────────┤
│ / filter  tab state  enter journey  esc back                                │
└──────────────────────────────────────────────────────────────────────────────┘
```

Journey (one patient, booking to payment):

```
┌ PATIENT P01136 · CT abdomen/pelvis with contrast · Urgent ────────────────────┐
│ Referral   Northside Primary Care (fictional)   Payer  Commercial            │
│                                                                              │
│ ✓ Ordered        Mon Oct 12 15:20                                            │
│ ✓ Booked         Mon Oct 12 15:20   for Tue 08:30 (lead 0.7 d)               │
│ ✓ Arrived        Tue 08:21          9m early                                 │
│ ✓ Registered     08:24–08:29        Jordan                                   │
│ ✓ Exam           08:41–09:09        Riley · room CT · 28m (slot 30m)         │
│   waited 20m from arrival, 12m after registration (tech shared with XR)      │
│ ✓ Report final   10:41              Dr. Avery · TAT 1h 32m (urgent ⚠ >60m)   │
│ ○ Claim          pending submission (charge lag)                             │
│ ○ Payment        —                                                           │
│                                                                              │
│ Expected net revenue $395 ($260 × 1.60 × NCR 0.95) · supplies $50 · PACS $3  │
├──────────────────────────────────────────────────────────────────────────────┤
│ esc back                                                                     │
└──────────────────────────────────────────────────────────────────────────────┘
```

Only observed facts appear (no "will be paid on…", no sampled durations before
they happen).

### 4.5 Staff

```
┌ STAFF · Tue Oct 13 ──────────────────────────────────────────────── 10:42 ───────────────────┐
│               08   09   10   11   12   13   14   15   16   17   18   19   20                  │
│ Jordan  Desk  ████████████████◐◐██████████████████                      ● registering        │
│ Taylor  Ops   ████████████████████████████████████                      ○ admin (covers desk)│
│ Riley   XR CT ████████████████◐◐██████████████████                      ● XR exam            │
│ Sam     CT XR                 ██████████████◐◐██████████████████████    ○ starts 11:30       │
│ Casey   MR    ████████████████████◐◐████████████████                    ● MR exam            │
│ Morgan  US          ██████████████████◐◐████████████████                ● US exam            │
│ Avery   Rad   ██████████████████████████████████            (60% site)  ● reading            │
│ Blake   Rad                   ██████████████████████████████████ (50%)  ○ starts 11:30       │
│                                                                                              │
│ COVERAGE   XR ███████████████████████████████████████████████ (shared 08–11:30, 16:15–20)    │
│            US ──────██████████████████████████████████──────────                             │
│            CT ███████████████████████████████████████████████ (shared)                       │
│            MR ████████████████████████████████████───────────────                            │
│            Desk ███████████████████████████████████▒▒▒▒──────── (▒ ops mgr, ─ techs self-reg)│
├─ SELECTED: Riley ────────────────────────────────────────────────────────────────────────────┤
│ Technologist · XR, CT · $36/h · week 15.5h of 40 · OT approved today 0m · breaks skipped 0   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ j/k select   o extend shift (overtime)   esc back                                            │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 4.6 Worklist (radiology)

```
┌ WORKLIST · 12 unread · 2 urgent · policy: urgent then FIFO ────────── 10:42 ───┐
│  PRI  STUDY   MODALITY  EXAM                AGE     STATUS                     │
│▶ ‼    S02201  CT        Chest w/ contrast    21m    unread                     │
│  ‼    S02198  US        Venous duplex         38m    reading · Dr. Avery 4m    │
│  R    S02150  MR        L-spine w/o        3h 12m    unread                     │
│  R    S02161  CT        Head w/o           2h 40m    unread                     │
│  R    S02170  XR        Chest 2V           1h 55m    unread                     │
│  …                                                                             │
│ Today  final 26 · TAT median 2h05 · p90 5h10 · urgent breaches 0               │
│ By modality  XR 3 · US 2 · CT 4 · MR 3 unread                                  │
├────────────────────────────────────────────────────────────────────────────────┤
│ j/k select  enter patient  P worklist policy  esc back                         │
└────────────────────────────────────────────────────────────────────────────────┘
```

### 4.7 Finance

```
┌ FINANCE · Tue Oct 13 ──────────────────────────────────────────────────── 10:42 ───┐
│                     TODAY        MONTH-TO-DATE     TRAILING 30 DAYS                 │
│ Net revenue        $11,940         $112,600          $261,500                       │
│ Operating expense   $8,615         $103,370          $239,400                       │
│   Payroll + benef.  $4,210          $45,300          $105,100                       │
│   Facility+equip.   $2,855          $37,100           $88,500                       │
│   Supplies, PACS    $1,290          $15,340           $32,700                       │
│   Billing fees        $260           $5,630           $13,100                       │
│ Operating profit    $3,325           $9,230           $22,100   margin 8.5%         │
│                                                                                     │
│ CASH            $142,850   ▼$1,940 today   (payroll Fri −$48,500)                   │
│ A/R             $412,300   days in A/R 46   denial rate 6.0%   collected 30d $262k  │
│                                                                                     │
│ Revenue/exam $158   Cost/exam $145   Revenue per busy scanner-hour: XR $255 US $320 CT $650 MR $500 │
│ Cash, last 30 days  ▁▂▃▃▄▄▅▅▆▅▄▃▄▅▆▆▇▆▅▄▃▄▅▅▆▇▇▆▅▅                                   │
├─────────────────────────────────────────────────────────────────────────────────────┤
│ esc back                                                                            │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

Shows the deliberate distinction: profit is positive while cash may fall;
upcoming known obligations (payroll date) are visible because a manager knows
them.

### 4.8 Event log

Full scrollable log with filters by category (`patients`, `rooms`, `staff`,
`reading`, `finance`, `alerts`) and severity. `enter` on an entry jumps to the
referenced patient / room / employee.

### 4.9 Actions menu (V0.1)

```
┌ ACTIONS ────────────────────────────────────────────────┐
│ 1  Extend a shift (approve overtime)…                    │
│ 2  Walk-in policy:   ( ) accept always                   │
│                      (•) accept if gap                   │
│                      ( ) schedule later                  │
│ 3  Worklist policy:  (•) urgent, then FIFO               │
│                      ( ) FIFO   ( ) oldest first          │
│                      ( ) shortest first                  │
├──────────────────────────────────────────────────────────┤
│ Result: ✓ Overtime approved: Morgan +60m today (≈$66)    │
└──────────────────────────────────────────────────────────┘
```

Every action shows its validation result and, where cheaply knowable, an
estimated cost. Rejections show the reason.

### 4.10 Daily summary (end of day)

Shown when `e` completes or the day ends with auto-pause on:

```
┌ END OF DAY · Tue Oct 13 ──────────────────────────────────────────────────┐
│ Exams 71 (XR 31 · US 12 · CT 18 · MR 10)    Walk-ins 8, left unseen 1     │
│ No-shows 4 · Cancellations 5 · Lost demand today 3 orders                 │
│ Wait median 9m · p90 34m · Late starts 14                                 │
│ Reports final 69 · TAT median 2h18 · urgent breaches 1 · unread at close 9│
│ Utilization (open hrs) XR 50% US 55% CT 55% MR 58%                        │
│ Utilization (staffed)  XR 50% US 84% CT 55% MR 88%                        │
│ Overtime 45m ($41) · skipped breaks 1 · desk unstaffed with patients 2h40 │
│ Revenue $15,880 · Expenses $13,420 · Operating profit $2,460              │
│                                                                           │
│ Insight: MR had 0 bookable slots after 16:30 while 6 MR orders were lost  │
│ to >21-day waits this week.                                               │
├───────────────────────────────────────────────────────────────────────────┤
│ enter continue to next day   f finance   esc dismiss                      │
└───────────────────────────────────────────────────────────────────────────┘
```

"Insight" lines are rule-based observations from observed metrics only — they
teach the player where the levers are. They never reveal hidden state.

### 4.11 Help / field guide

`?` shows key bindings for the current screen plus a short field guide: what
each metric means, how utilization is defined, why cash and profit diverge, and
that numbers are illustrative.

---

## 5. Model structure (code organization)

```
internal/tui/
  app.go        root model: screens, global keys, tick, pacing, session calls
  pacing.go     speed table, frame budget, auto-pause settings
  layout.go     breakpoints, panel composition helpers
  theme.go      palette, glyphs, NO_COLOR handling
  dashboard.go  one file per screen: model fragment + view
  schedule.go
  patients.go
  staff.go
  worklist.go
  finance.go
  eventlog.go
  actions.go
  summary.go
  help.go
  format.go     money, durations, times, sparkline
  *_test.go     update tests + golden views
```

Each screen is a small struct with `Update(msg, obs) (screen, cmd)` and
`View(obs, w, h) string`. The root model owns the session pointer and passes
the cached observation down. No screen mutates the observation.

---

## 6. Testing the TUI

* **Golden views**: construct a session from a fixed scenario+seed, advance to
  fixed times, render each screen at 80×24, 120×36, 160×48 with colors
  disabled; compare to files under `internal/tui/testdata/`. Regenerate with
  `-update`.
* **Update tests**: key sequences produce expected UI state and session calls
  (e.g., `space` toggles running; `n` lands on the next observable entry).
* **Pacing test**: drive `tickMsg`s with a fake wall clock at different frame
  rates and speeds; final digests must match the headless run with the same
  action log.
* **Manual smoke checklist** in ROADMAP for each release (resize, NO_COLOR,
  small terminal, Max speed responsiveness on the Raspberry Pi).
