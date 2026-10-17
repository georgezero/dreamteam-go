package eval

// RecordSchema names the per-seed measurement record: what a harness
// observes of one plan on one seed. Times are simulation seconds.
const RecordSchema = "dreamteam.evalrecord/v1"

// Record is one plan's run on one seed, from the takeover until
// ObservedUntil. Studies and orders are every one the window can see:
// studies unfinished at the takeover (inherited) and everything after.
type Record struct {
	Schema              string           `json:"schema"`
	Plan                string           `json:"plan"`
	PlanSHA256          string           `json:"planSHA256"`
	Seed                int64            `json:"seed"`
	Engine              string           `json:"engine"`
	ScenarioFingerprint string           `json:"scenarioFingerprint"`
	TATTargetMin        map[string]int64 `json:"tatTargetMin"`
	TakeoverSec         int64            `json:"takeoverSec"`
	TakeoverDate        string           `json:"takeoverDate"` // practice-local
	// LastActionSec: the latest accepted batch (0: none). It must be
	// strictly before the primary endpoint (V0.2_DESIGN §13.1).
	LastActionSec int64 `json:"lastActionSec"`
	// LatestEffectSec: the latest accepted effect (hire productive,
	// termination end, roster/allocation version, contract start or end).
	LatestEffectSec  int64 `json:"latestEffectSec"`
	ObservedUntilSec int64 `json:"observedUntilSec"`
	// TeleOnlyOf names the plan this one differs from only by
	// teleradiology actions: their bookings must be identical.
	TeleOnlyOf         string `json:"teleOnlyOf,omitempty"`
	AppointmentsSHA256 string `json:"appointmentsSHA256"`

	Studies  []Study   `json:"studies"`
	Orders   []Order   `json:"orders"`
	Samples  []Sample  `json:"samples"`  // daily, d = 1.., at the takeover's clock time
	Captures []Capture `json:"captures"` // at horizon instants, and "end"
}

// Study is one study's timestamps (0: not yet).
type Study struct {
	ID           string `json:"id"`
	Priority     string `json:"priority"`
	Modality     string `json:"modality"`
	AcquiredSec  int64  `json:"acquiredSec"`
	EnteredSec   int64  `json:"enteredSec"`
	ClaimedSec   int64  `json:"claimedSec"`
	ReadStartSec int64  `json:"readStartSec"` // remote: handoff done; local: the claim
	FinalSec     int64  `json:"finalSec"`
	Remote       bool   `json:"remote"`
	FeeCents     int64  `json:"feeCents"` // remote: fee and surcharge locked at claim
}

// Order is one referral or walk-in.
type Order struct {
	ID          string `json:"id"`
	Priority    string `json:"priority"`
	Modality    string `json:"modality"`
	WalkIn      bool   `json:"walkIn"`
	ReceivedSec int64  `json:"receivedSec"`
	// WantedSec: the original wanted service time (the earliest day the
	// patient asked for, at its start; a walk-in's arrival). A later
	// promise never moves it.
	WantedSec     int64 `json:"wantedSec"`
	FirstApptSec  int64 `json:"firstApptSec"` // start of the first appointment booked (0: none)
	Displacements int   `json:"displacements"`
}

// Order outcomes, as of a capture.
const (
	OutServed       = "served"        // imaged (reported or not)
	OutLost         = "lost"          // practice-caused: declined, no slot, displaced and lost
	OutLeft         = "left"          // walk-in abandonment
	OutPatient      = "patient"       // patient cancellation or no-show
	OutOpen         = "open"          // booked, on the recall list or in a visit
	OutNotDisclosed = "not_disclosed" // received after the capture
)

// Sample is the backlog and money at one daily instant. Stages count
// unfinished studies (acquired, not final), inherited included.
type Sample struct {
	Day           int    `json:"day"`
	AtSec         int64  `json:"atSec"`
	Date          string `json:"date"`
	AwaitingDoc   int    `json:"awaitingDocumentation"`
	Unread        int    `json:"unread"`
	RemoteHandoff int    `json:"remoteHandoff"`
	LocalReading  int    `json:"localReading"`
	RemoteReading int    `json:"remoteReading"`
	Overdue       int    `json:"overdue"` // unfinished past their TAT target
	AgeP50Min     int64  `json:"ageP50Min"`
	AgeP90Min     int64  `json:"ageP90Min"`
	AgeMaxMin     int64  `json:"ageMaxMin"`
	AcquiredCum   int    `json:"acquiredCum"` // since the takeover
	FinalCum      int    `json:"finalCum"`
	ProfitCents   int64  `json:"profitCents"` // evaluation operating profit, cumulative
	CashCents     int64  `json:"cashCents"`
}

// Backlog is the unfinished total of a sample.
func (s Sample) Backlog() int {
	return s.AwaitingDoc + s.Unread + s.RemoteHandoff + s.LocalReading + s.RemoteReading
}

// Capture is the state at one horizon instant.
type Capture struct {
	Name          string   `json:"name"` // T30, T90, T180, tailEnd, end
	AtSec         int64    `json:"atSec"`
	OrderOutcomes []string `json:"orderOutcomes"` // aligned with Record.Orders
	Money         Money    `json:"money"`
}

// Money is the evaluation window from the takeover to a capture, and the
// balances and commitments at it. Commitments are not expenses: they are
// listed beside the accrual P&L, never added to it.
type Money struct {
	NetRevenueCents       int64 `json:"netRevenueCents"`
	OperatingExpenseCents int64 `json:"operatingExpenseCents"`
	OperatingProfitCents  int64 `json:"operatingProfitCents"`
	CashCents             int64 `json:"cashCents"`
	// MinCashCents: the lowest cash after any posting since the takeover
	// (every transaction, not a daily sample), and when.
	MinCashCents          int64 `json:"minCashCents"`
	MinCashSec            int64 `json:"minCashSec"`
	ReceivablesCents      int64 `json:"receivablesCents"`
	PayablesCents         int64 `json:"payablesCents"`
	AccruedPayrollCents   int64 `json:"accruedPayrollCents"`
	SeverancePayableCents int64 `json:"severancePayableCents"`
	// Earned but not yet posted: hourly pay of shifts in progress.
	EarnedUnpostedWagesCents int64 `json:"earnedUnpostedWagesCents"`
	// Remote: fees committed by claims not yet final, and the minimum
	// retainers still owed under accepted contracts (not yet accrued).
	RemoteInFlightCents    int64 `json:"remoteInFlightCents"`
	ContractMinimumCents   int64 `json:"contractMinimumCents"`
	UnbilledCompletedCents int64 `json:"unbilledCompletedCents"` // imaged, not reported: future revenue, not cash
	PastDueCents           int64 `json:"pastDueCents"`
	CashBridgeReconciles   bool  `json:"cashBridgeReconciles"`
	// Ledger expense since the takeover: remote retainer and read fees,
	// and what the vendor was paid.
	RemoteRetainerCents int64 `json:"remoteRetainerCents"`
	RemoteReadsCents    int64 `json:"remoteReadsCents"`
	VendorPaidCents     int64 `json:"vendorPaidCents"`
}
