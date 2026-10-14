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

As built in V0.1 step 6 (before arrivals exist): `░` booked, `·` open,
`˙` free but too late to book, `─` unstaffed, `◐` the room's tech on a
planned break, `⊘` contrast slot with no radiologist on site. The list shows
every slot of the selected room. Runs of slots that can't be booked collapse
into one line that gives the reason. The header compares the modality's
demand today (orders, booked, lost) with its next open slot. Paging runs
from the start date to the end of the booking horizon. The dashboard gains a
**DEMAND** panel per modality. It shows today's orders, bookings, recalls,
losses and cancellations, today's appointments and open slots, and the next
open slot, in amber when it is more than 14 days out. The dashboard log shows
at most 8 entries. `l` opens the full retained log.

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

**Closing status** (step 8, SIMULATION_DESIGN §6.9). The practice line
shows the day's phase, not just open/closed:

```
● OPEN 08:00–20:00
◐ CLOSING — 3 patients remaining · OT +42m after 20:00 ($38) · Sam, Dr. Blake held over
○ CLOSED · operations closed 20:47 (scheduled 20:00) · OT 1h 12m ($63)
```

After scheduled close the room panel shows only rooms with admitted
patients or an exam in progress; the others read `Closed`. Log entries
(amber): "Sam stays past shift end: 3 patients still need CT/XR", "P00123
rescheduled: no MR technologist can finish within the 12-hour limit". Info
entries: "Sam off duty after 42m overtime", "Operations closed 20:47". The
staff list marks held-over staff `OT +42m`.

As built in V0.1 step 7, `p` opens the list for a day: every appointment
booked for it (time order, cancelled included) and every walk-in. Arrival
shows the actual time and, when the patient was more than 5 minutes off, the
offset. Waited runs to now, or to when the visit ended. `tab` cycles the
filters: all, waiting, missed / no-show, left / unseen, walk-ins. `←→` pages
days and `enter` opens the journey: ordered (or walked in), booked, missed
(provisional), arrived and how early or late, waiting, left, sent home,
no-show, rescheduled, lost. Nothing is shown before it happens: no expected
arrival, no-show odds or patience. The dashboard has a **PATIENT FLOW**
panel. It shows who is waiting now by modality and the longest wait, the
no-shows at the last close, and today's arrivals, late arrivals, walk-ins,
walk-ins who left, provisional misses and patients sent home unseen. Each
room line shows how many patients are waiting for it. The top bar shows the
version (`DREAMTEAM v0.1.0-dev.N`).

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

As built in V0.1 step 9, `w` opens the worklist. The header gives unread,
reading, urgent and the policy, then the turnaround line, then backlog by
modality and priority with the oldest study. The rows list reading studies
first, then unread in the order the policy offers them. Each row shows
priority (‼ urgent, amber soon), study, patient, modality, exam, time since
imaging complete (amber past target) and status (`reading · Dr. Avery 4m`).
Below is **LATEST REPORTS** with TAT, marked when over target. `enter`
opens the patient's journey, which now runs imaging complete → on worklist
→ being read / read → report final (TAT, against target). The dashboard
adds one reporting line: unread (‼ urgent), reading, oldest, final today,
TAT median and p90 today, breaches. The staff panel shows radiologists as
reading (with the study), ready to read, other duties, supervising P… contrast,
or off.

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

As built (V0.1 step 10):
* `f` opens the screen. `tab` cycles three views, and `↑↓` scroll:
  * **overview**: columns for today, month to date, trailing 30 days and
    since day 1. Rows: revenue earned, collection variance, net revenue,
    operating expense (labor, fixed, supplies/contrast/PACS, billing fees),
    operating profit and margin, cash collected, cash paid out, net cash
    flow and reports final. Below the table:
    * cash with today's change, and the next payroll and bills with what
      has accrued so far;
    * A/R, days in A/R, the 30-day denial rate, open claims by state, A/R
      aging and closed claims by outcome;
    * revenue and direct cost by modality over 30 days;
    * a 30-day cash sparkline.
  * **claims**: open claims, oldest first.
  * **journal**: the latest entries with their debit and credit lines.
* Money is whole dollars, and negative amounts are red with a "−".
  Undefined margins show "—".
* The dashboard's second line is the compact money summary: cash and
  today's change, revenue, expenses and operating profit today, and A/R.
* Patient journeys continue past the report: revenue earned, claim
  submitted, then denied or resubmitted, then paid, short paid or written
  off.

### 4.7a Opening briefing and daily history (V0.1 step 11)

The established practice opens on a **takeover briefing**, built from the
180-day warm-up, which runs before the TUI starts and is never shown
advancing. It covers:
* **Money:** opening cash and the owner capital added at takeover, with
  what the history left; the last 30 days' net revenue, expenses,
  operating profit and margin; cash collected and paid out.
* **Carried over:** the history's result, receivables and open claims,
  payables and payroll owed; the next payroll and bills.
* **Financing:** a note that no credit line is modelled and that negative
  cash is tracked as a shortfall.
* **Patients:** the last 7 days' orders, exams, lost orders and no-shows;
  median wait; room utilization of open and of staffed hours; appointments
  booked ahead and the recall list.
* **Reporting:** the worklist now and the recent TAT.
* **Today:** opening hours, appointments, and the rostered staff with
  their shifts.

`enter` goes to the dashboard, `space` starts the day, and `b` reopens the
briefing. `h` shows the **daily history**: one row per completed day, with
history days marked "·" and dimmed. Columns: orders, exams, no-shows,
lost, median wait, TAT, unread at close, utilization by modality, net
revenue, expenses, profit and **actual** end-of-day cash. A takeover line
names the owner capital and the cash before and after it; the finance
chart is "Cash (actual)" with the capital noted at takeover. The balance
line shows exact cents, so its equation holds as printed.

The finance overview adds:
* the balance sheet (assets = liabilities + equity, with equity split into
  opening capital, result before day 1 and result since day 1);
* the financing line (lowest cash since day 1, days that ended below zero,
  and a red **FINANCING SHORTFALL** while cash is negative);
* the revenue rows, now "Revenue recognized", "+ collected vs expected" and
  "Net revenue", with a note on the difference.

The dashboard money line flags "⚠ shortfall".

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

**As built (V0.1 step 13).** `a` opens the actions screen (auto-pause moved
to `A`). One list: the employees rostered on the chosen day (`[` `]` pick
today or a day up to two weeks ahead) with their shift, 12-hour cap, status
today and any approved extension; then the walk-in and worklist policy
rows. `↑↓` choose a row, `←→` change it (±15 minutes, or the next policy),
`enter` submits through the same `Apply` an agent uses. The cursor row is
previewed live ("Preview ✓ Shift extended: Morgan today until 19:15 (+1h
15m approved overtime · ≈ cost)" or "Preview ✗ … past the 12-hour limit at
21:30 (at most +2h 30m more)"); the result and the latest actions (ID, time,
✓/✗) follow. The dashboard shows the policies in force in the patient-flow
header, "+1h 15m approved" in the staff panel and "+ext" in the staff
summary. Fits 80×24.

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

**As built (V0.1 step 12).** `e` (from any screen) runs today's operations
to operational close and opens the summary. Below 110 columns every line has
a short form, so the day fits 80×24 untruncated:

```
─ END OF DAY · Tue Oct 13 · so far, to 21:28 · closed 21:28 ──────────────
 Takeover day: from midnight, incl. before 06:00 · tab: since takeover
 Patients  70 exams (XR 29 US 9 CT 24 MR 8) · walk-ins 7, left 4
 Demand    63 orders · lost 4 (6%) · no-shows 3% · slot fill 77%
 Waits     arrival→exam p50 17m · p90 44m · over 30m 11
 Rooms     busy/open    XR 41% US 48% CT 53% MR 49%
           busy/staffed XR 42% US 72% CT 56% MR 73%
           staffed/open XR 96% US 67% CT 96% MR 67%
 Staff     techs busy 87% · reading 76% of blocks · overtime 3h 01m ($65)
 Reporting final 62 · TAT p50 29m · p90 1h 15m · unread at close 8
 Money     revenue $9,210 · expenses $8,635 · profit $575 (6.2%)
 Cash      in $6,239 · out $0 · cash $156,239 · A/R $281,125
 Alerts    16 raised (4 urgent)
```

`←→` page through earlier days (completed days are final; today says "so
far"), `t` returns to today, `tab` switches to **since takeover**: the
inherited baseline, operating results (excluding inherited-claim variance,
including the takeover day's accruals after the takeover), cash flow (with
the inherited part), the change in A/R, A/P and payroll owed, a ✓ when the
cash bridge reconciles, and the window's operations. `enter` returns to the
dashboard.

The dashboard adds two things and gives up one: a **Today** line
(utilization of staffed hours by room, time-weighted, and the arrival→exam
wait so far) and an **ALERTS** panel (active alerts, most severe first, with
the time raised, the screen key that shows more, and a "why:" line for the
first one, built from observed facts). On screens under 28 rows the demand
table shrinks to one line so the alerts fit.

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
