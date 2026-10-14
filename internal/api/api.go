// Package api is the public contract between the simulation and everything
// that drives or displays it: the TUI, the future WebUI, CLI summaries and
// agents. It contains only what a practice manager could know. Hidden
// simulator state never appears here.
//
// The types grow as screens need them (ROADMAP §2, step 4 onward).
package api

import "github.com/georgezero/dreamteam-go/internal/simtime"

// ObservationSchema identifies the observation format.
const ObservationSchema = "dreamteam.observation/v1"

// Severity classifies log entries. Alerts may auto-pause play.
type Severity string

const (
	Info  Severity = "info"
	Warn  Severity = "warn"
	Alert Severity = "alert"
)

// LogEntry is one observable event-log record.
type LogEntry struct {
	Seq      uint64       `json:"seq"`
	At       simtime.Time `json:"-"`
	Time     string       `json:"time"` // practice-local ISO-8601, no zone
	Severity Severity     `json:"severity"`
	Category string       `json:"category"`
	Text     string       `json:"text"`
}

// Practice is the practice-level status.
type Practice struct {
	Name     string `json:"name"`
	Open     bool   `json:"open"`
	Scenario string `json:"scenario"`
	Hours    string `json:"hours"` // today's open hours, "08:00–20:00"; empty when closed all day
	// BookingHorizonDays is how far ahead appointments can be booked.
	BookingHorizonDays int64 `json:"bookingHorizonDays"`
	// StartOffset is the start date relative to today, in days (≤ 0).
	StartOffset int64 `json:"startOffset"`

	// Phase is "open" (scheduled hours), "closing" (after scheduled close
	// while admitted patients are still being served) or "closed".
	Phase          string `json:"phase"`
	ScheduledClose string `json:"scheduledClose,omitempty"` // "20:00"
	// OperationalClose is when today's operations actually ended, once
	// they have ("20:47").
	OperationalClose string `json:"operationalClose,omitempty"`
	OnSite           int    `json:"onSite"` // admitted patients still in the building
	// Overtime today: minutes past scheduled shift ends (running ones
	// included) and the cost booked so far (cents, when staff go home).
	OvertimeMin       int64    `json:"overtimeMin"`
	OvertimeCostCents int64    `json:"overtimeCostCents"`
	HeldOver          []string `json:"heldOver,omitempty"` // names of staff kept past shift end
}

// Practice phases.
const (
	PhaseOpen    = "open"
	PhaseClosing = "closing"
	PhaseClosed  = "closed"
)

// Room states (SIMULATION_DESIGN §2.5).
const (
	RoomClosed    = "closed"
	RoomUnstaffed = "unstaffed"
	RoomIdle      = "idle"
	RoomOccupied  = "occupied"
)

// RoomExam is the exam in progress in a room. Expected is the catalog
// median (published); the sampled duration is never shown.
type RoomExam struct {
	Visit        string `json:"visit"`
	Patient      string `json:"patient"`
	ExamName     string `json:"examName"`
	Technologist string `json:"technologist"`
	StartedAt    string `json:"startedAt"`
	ElapsedMin   int64  `json:"elapsedMin"`
	ExpectedMin  int64  `json:"expectedMin"`
}

// Room is one room's observable status.
type Room struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Modality string   `json:"modality"`
	State    string   `json:"state"`
	StaffBy  []string `json:"staffedBy"` // employee IDs
	// Shared is true when a technologist staffing this room also staffs
	// another one.
	Shared  bool      `json:"shared"`
	Current *RoomExam `json:"current,omitempty"`
}

// Employee statuses.
const (
	StaffOff         = "off"
	StaffBusy        = "busy"
	StaffOnDuty      = "on_duty"
	StaffOnBreak     = "on_break"
	StaffReading     = "reading"      // radiologist reading a study now
	StaffOtherDuties = "other_duties" // radiologist on shift, outside site reading blocks
	StaffReadingFree = "read_ready"   // radiologist in a site block, not reading (worklist empty or nothing fits)
)

// Employee is one staff member's observable status.
type Employee struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Role     string   `json:"role"`
	Skills   []string `json:"skills"`
	Status   string   `json:"status"`
	Shift    string   `json:"shift"`              // today's shift "07:45–16:15", empty when off today
	Remote   bool     `json:"remote,omitempty"`   // radiologist working remotely today
	SiteBP   int64    `json:"siteBP,omitempty"`   // radiologist site allocation
	Rooms    []string `json:"rooms,omitempty"`    // rooms this technologist staffs now
	StartsAt string   `json:"startsAt,omitempty"` // shift start, while not yet on duty
	EndsAt   string   `json:"endsAt,omitempty"`   // shift or break end, while on duty or break
	Task     string   `json:"task,omitempty"`     // while busy: "registering P00042", "XR exam P00042"
	// Held is true while kept past the scheduled shift end because patients
	// still need them; OvertimeMin is today's time past shift end so far.
	Held        bool  `json:"held,omitempty"`
	OvertimeMin int64 `json:"overtimeMin,omitempty"`
	// Supervising names the patient whose contrast exam this radiologist
	// is supervising now.
	Supervising string `json:"supervising,omitempty"`
	// Extensions (action extend_shift): the rostered end, the minutes
	// approved today and the latest the shift could end (start + the
	// shift cap). Pay: hourly rate (0 for salaried) and paid minutes this
	// week before today, for overtime estimates.
	RosterEnd     string `json:"rosterEnd,omitempty"`
	ApprovedMin   int64  `json:"approvedMin,omitempty"`
	CapAt         string `json:"capAt,omitempty"`
	HourlyCents   int64  `json:"hourlyCents,omitempty"`
	WeekPaidMin   int64  `json:"weekPaidMin"`
	ApprovedOTMin int64  `json:"approvedOTMin,omitempty"` // of OvertimeMin, approved in advance
}

// Desk coverage levels.
const (
	DeskClosed  = "closed"
	DeskStaffed = "staffed" // front desk
	DeskOps     = "ops"     // operations manager covering
	DeskSelf    = "self"    // technologists self-register
)

// Desk is who registers patients right now.
type Desk struct {
	Coverage string `json:"coverage"`
	By       string `json:"by,omitempty"` // employee ID
}

// Observation is a snapshot of everything currently observable.
type Observation struct {
	Schema   string       `json:"schema"`
	Now      simtime.Time `json:"-"`
	Time     string       `json:"time"`
	Day      int64        `json:"day"` // days since the start date, 1-based
	Weekday  string       `json:"weekday"`
	Date     string       `json:"date"`
	Clock    string       `json:"clock"`
	Practice Practice     `json:"practice"`
	Rooms    []Room       `json:"rooms"`
	Staff    []Employee   `json:"staff"`
	Desk     Desk         `json:"desk"`
	Demand   Demand       `json:"demand"`
	Flow     Flow         `json:"flow"`
	Worklist Worklist     `json:"worklist"`
	Finance  Finance      `json:"finance"`
	// Today is today's operational metrics so far (partial: Complete is
	// false until the day ends).
	Today    PeriodMetrics `json:"today"`
	Alerts   Alerts        `json:"alerts"`
	Policies Policies      `json:"policies"`
	Actions  ActionLog     `json:"actions"`
	// Extensions lists the shift extensions approved for today and later.
	Extensions []Extension `json:"extensions"`
	// LogTotal is the number of log entries ever emitted.
	LogTotal  uint64     `json:"logTotal"`
	RecentLog []LogEntry `json:"recentLog"` // oldest first
}

// DemandCounts counts demand events: orders received, appointments booked
// (including recall releases and rebookings), orders put on the recall
// list, orders lost (declined or no capacity) and appointments cancelled.
type DemandCounts struct {
	Received  int `json:"received"`
	Booked    int `json:"booked"`
	Recall    int `json:"recall"`
	Lost      int `json:"lost"`
	Cancelled int `json:"cancelled"`
}

// ModalityDemand is demand and capacity for one modality.
type ModalityDemand struct {
	Modality   string       `json:"modality"`
	Today      DemandCounts `json:"today"`
	SinceStart DemandCounts `json:"sinceStart"`
	ApptsToday int          `json:"apptsToday"` // appointments on today's schedule
	OpenToday  int          `json:"openToday"`  // free bookable slots still bookable today
	// NextOpen is the date of the earliest free slot a new order could book
	// now ("" when none within the booking horizon); NextOpenDays counts
	// days from today.
	NextOpen     string `json:"nextOpen"`
	NextOpenDays int    `json:"nextOpenDays"`
}

// Demand summarizes referral demand against booking capacity.
type Demand struct {
	Modalities []ModalityDemand `json:"modalities"`
	RecallList int              `json:"recallList"` // orders waiting for their date to enter the horizon
}

// Span is a time-of-day interval [From, To), in seconds since midnight.
type Span struct {
	From simtime.Duration `json:"from"`
	To   simtime.Duration `json:"to"`
}

// Slot statuses.
const (
	SlotOpen          = "open"           // free and bookable
	SlotPassed        = "passed"         // free, but too soon or past to book
	SlotBooked        = "booked"         // holds an appointment
	SlotUnstaffed     = "unstaffed"      // no technologist rostered for the whole slot
	SlotNoSupervision = "no_supervision" // contrast slot without an on-site radiologist
	SlotSharedBusy    = "shared_busy"    // its shared technologist is booked in another room then
)

// Appointment is one booked appointment.
type Appointment struct {
	ID       string `json:"id"`
	Order    string `json:"order"`
	Patient  string `json:"patient"`
	Exam     string `json:"exam"`
	ExamName string `json:"examName"`
	Priority string `json:"priority"`
	Source   string `json:"source"`
	BookedAt string `json:"bookedAt"`
}

// Slot is one template slot on a given day.
type Slot struct {
	Span
	Category string       `json:"category"`
	Status   string       `json:"status"`
	Appt     *Appointment `json:"appointment,omitempty"`
}

// RoomSchedule is one room's day: planned staffing and slots.
type RoomSchedule struct {
	Room     string `json:"room"`
	Name     string `json:"name"`
	Modality string `json:"modality"`
	Staffed  []Span `json:"staffed"` // open-hours intervals with a technologist rostered
	Breaks   []Span `json:"breaks"`  // unstaffed only because the rostered tech is on break
	Slots    []Slot `json:"slots"`
}

// DaySchedule is the schedule of one day.
type DaySchedule struct {
	Date    string         `json:"date"`
	Weekday string         `json:"weekday"`
	Offset  int64          `json:"offset"` // days from today
	Hours   *Span          `json:"hours"`  // nil when closed
	Rooms   []RoomSchedule `json:"rooms"`
}

// FlowCounts count patient-flow events.
type FlowCounts struct {
	Arrived         int `json:"arrived"`         // scheduled patients who arrived
	Early           int `json:"early"`           // … more than 5 min early
	Late            int `json:"late"`            // … more than 5 min late
	AfterGrace      int `json:"afterGrace"`      // … after being marked missed (fitted in)
	WalkIns         int `json:"walkIns"`         // walk-ins who came in
	WalkInsAdmitted int `json:"walkInsAdmitted"` // … taken in on that visit
	// WalkInsOffered counts walk-ins offered an appointment instead;
	// WalkInsDeclined, of them, those turned away by the walk-in policy
	// (the rest could not be served today). An offer either booked or
	// found no acceptable slot (the order is lost).
	WalkInsOffered     int `json:"walkInsOffered"`
	WalkInsDeclined    int `json:"walkInsDeclined"`
	WalkInsOfferBooked int `json:"walkInsOfferBooked"`
	WalkInsOfferFailed int `json:"walkInsOfferFailed"`
	Missed             int `json:"missed"`      // appointments marked missed at start + grace
	NoShows            int `json:"noShows"`     // missed appointments finalized at accounting close
	Rescheduled        int `json:"rescheduled"` // too late for the late policy, or after closing
	Left               int `json:"left"`        // walk-ins who left without an exam

	Registered     int `json:"registered"`
	SelfRegistered int `json:"selfRegistered"` // by a technologist (no desk staff on duty)
	ExamsStarted   int `json:"examsStarted"`
	ExamsDone      int `json:"examsDone"`
	AfterHours     int `json:"afterHours"` // exams started after scheduled close
	// NoCoverage counts admitted patients rescheduled because nobody could
	// serve them safely today (each with a recorded reason).
	NoCoverage        int   `json:"noCoverage"`
	HeldOver          int   `json:"heldOver"` // staff kept past shift end
	SkippedBreaks     int   `json:"skippedBreaks"`
	OvertimeMin       int64 `json:"overtimeMin"`       // booked when staff go home
	OvertimeCostCents int64 `json:"overtimeCostCents"` // hourly staff only
	// Of the overtime, the part approved in advance (extend_shift); the
	// rest was required hold-over.
	ApprovedOvertimeMin       int64 `json:"approvedOvertimeMin"`
	ApprovedOvertimeCostCents int64 `json:"approvedOvertimeCostCents"`
}

// Visit states for patients on site.
const (
	VisitArrived     = "arrived"     // waiting to register
	VisitRegistering = "registering" // at the desk
	VisitWaiting     = "waiting"     // registered, waiting for the exam
	VisitInExam      = "in_exam"
)

// WaitingPatient is a patient on site: arrived, registering, waiting or in
// an exam.
type WaitingPatient struct {
	State       string `json:"state"`
	Visit       string `json:"visit"`
	Patient     string `json:"patient"`
	Order       string `json:"order"`
	Appointment string `json:"appointment,omitempty"`
	Modality    string `json:"modality"`
	Exam        string `json:"exam"`
	ExamName    string `json:"examName"`
	Priority    string `json:"priority"`
	WalkIn      bool   `json:"walkIn"`
	ApptTime    string `json:"apptTime,omitempty"` // "10:15"
	Arrived     string `json:"arrived"`            // "10:02"
	WaitedMin   int64  `json:"waitedMin"`
	// LateMin is minutes after the appointment start (negative: early).
	LateMin    int64 `json:"lateMin"`
	AfterGrace bool  `json:"afterGrace"`
}

// Flow is the patient flow today.
type Flow struct {
	Today      FlowCounts `json:"today"`
	SinceStart FlowCounts `json:"sinceStart"`
	// MissedNow counts today's appointments currently missed: provisional,
	// they become no-shows at accounting close unless the patient comes.
	MissedNow        int              `json:"missedNow"`
	NoShowsLastClose int              `json:"noShowsLastClose"`
	Waiting          []WaitingPatient `json:"waiting"` // everyone on site, in arrival order
}

// Patient status keys for lists.
const (
	PatientBooked      = "booked"
	PatientCancelled   = "cancelled"
	PatientArrived     = "arrived" // waiting to register
	PatientRegistering = "registering"
	PatientWaiting     = "waiting"
	PatientInExam      = "in_exam"
	PatientDone        = "done"
	PatientMissed      = "missed"
	PatientNoShow      = "no_show"
	PatientRescheduled = "rescheduled"
	PatientLeft        = "left"
	PatientSentToBook  = "booked_later" // walk-in given an appointment instead
	PatientLost        = "lost"
)

// PatientRow is one patient on a day's list: an appointment for that day or
// a walk-in that day.
type PatientRow struct {
	Patient     string `json:"patient"`
	Order       string `json:"order"`
	Appointment string `json:"appointment,omitempty"`
	Visit       string `json:"visit,omitempty"`
	Modality    string `json:"modality"`
	ExamName    string `json:"examName"`
	Priority    string `json:"priority"`
	WalkIn      bool   `json:"walkIn"`
	ApptTime    string `json:"apptTime,omitempty"`
	Arrived     string `json:"arrived,omitempty"`
	Punctuality string `json:"punctuality,omitempty"` // "12m early", "on time", "9m late"
	WaitedMin   int64  `json:"waitedMin"`             // so far, or until the visit ended
	Status      string `json:"status"`                // a Patient* key
	StatusText  string `json:"statusText"`
}

// JourneyStep is one observed step of a patient's journey.
type JourneyStep struct {
	Time   string `json:"time"` // practice-local ISO-8601; empty when unknown
	Label  string `json:"label"`
	Detail string `json:"detail"`
	Done   bool   `json:"done"` // false for a step still ahead (a pending appointment)
}

// Journey is one patient's observable history.
type Journey struct {
	Patient  string        `json:"patient"`
	Order    string        `json:"order"`
	Exam     string        `json:"exam"`
	ExamName string        `json:"examName"`
	Modality string        `json:"modality"`
	Priority string        `json:"priority"`
	Source   string        `json:"source"`
	WalkIn   bool          `json:"walkIn"`
	Payer    string        `json:"payer"` // payer class, known at booking
	Status   string        `json:"status"`
	Steps    []JourneyStep `json:"steps"`
}

// Study states.
const (
	StudyAcquired = "acquired" // imaging complete, documentation in progress
	StudyUnread   = "unread"   // awaiting interpretation
	StudyReading  = "reading"
	StudyFinal    = "final"
)

// WorklistItem is one study.
type WorklistItem struct {
	Study       string `json:"study"`
	Patient     string `json:"patient"`
	Modality    string `json:"modality"`
	ExamName    string `json:"examName"`
	Priority    string `json:"priority"`
	State       string `json:"state"`
	Acquired    string `json:"acquired"` // practice-local ISO time imaging completed
	AgeMin      int64  `json:"ageMin"`   // since imaging completed (to final, once final)
	Radiologist string `json:"radiologist,omitempty"`
	ReadingMin  int64  `json:"readingMin,omitempty"` // elapsed, while reading
	TATMin      int64  `json:"tatMin,omitempty"`     // once final
	Overdue     bool   `json:"overdue"`              // past its TAT target
}

// Backlog counts unread and reading studies.
type Backlog struct {
	Key     string `json:"key"` // modality or priority
	Unread  int    `json:"unread"`
	Reading int    `json:"reading"`
}

// Worklist is the radiology worklist and report turnaround.
type Worklist struct {
	Policy  string         `json:"policy"`
	Unread  int            `json:"unread"`
	Reading int            `json:"reading"`
	Urgent  int            `json:"urgent"` // unread or reading
	Items   []WorklistItem `json:"items"`  // unread and reading, in the order the policy offers them (capped)
	Recent  []WorklistItem `json:"recent"` // latest finalized reports, newest first (capped)
	// OldestMin is the age of the oldest unfinished study since imaging.
	OldestMin   int64     `json:"oldestMin"`
	ByModality  []Backlog `json:"byModality"`
	ByPriority  []Backlog `json:"byPriority"`
	FinalToday  int       `json:"finalToday"`
	FinalTotal  int       `json:"finalTotal"`
	BreachToday int       `json:"breachToday"` // TAT targets missed today (urgent and soon when they pass; routine at finalization)
	// TAT over the studies acquired today, imaging complete → report final,
	// unfinished ones counted at their current age (censored, §11).
	AcquiredToday int   `json:"acquiredToday"`
	TATMedianMin  int64 `json:"tatMedianMin"`
	TATP90Min     int64 `json:"tatP90Min"`
	// Median worklist wait (entered → claimed) and read time (claimed →
	// final) over reports finalized today.
	WaitMedianMin int64 `json:"waitMedianMin"`
	ReadMedianMin int64 `json:"readMedianMin"`
	// TAT medians by priority over studies acquired today (censored).
	TATByPriority []PriorityTAT `json:"tatByPriority"`
	TargetMin     []PriorityTAT `json:"targetMin"` // the TAT targets
}

// PriorityTAT is a per-priority figure.
type PriorityTAT struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
	Minutes  int64  `json:"minutes"`
}

// Claim states. Closed claims (paid, short_paid, written_off) carry no
// balance.
const (
	ClaimPending     = "pending"     // report final, awaiting submission (charge lag)
	ClaimSubmitted   = "submitted"   // with the payer (self-pay: statement sent)
	ClaimDenied      = "denied"      // denied, awaiting resubmission
	ClaimResubmitted = "resubmitted" // reworked and resubmitted
	ClaimPaid        = "paid"
	ClaimShortPaid   = "short_paid"
	ClaimWrittenOff  = "written_off"
)

// Finance is the practice's money: balances, period statements, claims and
// the latest journal entries. Amounts are integer cents. Revenue is earned
// when a report is finalized (accrual); cash moves only on payments
// received and bills or payroll paid, so profit and cash differ.
type Finance struct {
	CashCents int64 `json:"cashCents"`
	// CashChangeTodayCents is today's operating cash flow: collections
	// minus disbursements (the opening adjustment is not a cash flow).
	CashChangeTodayCents int64 `json:"cashChangeTodayCents"`
	ARCents              int64 `json:"arCents"`             // accounts receivable: expected net still owed on open claims
	APCents              int64 `json:"apCents"`             // accounts payable: accrued bills not yet paid
	AccruedPayrollCents  int64 `json:"accruedPayrollCents"` // wages and benefits earned, not yet paid
	// Periods: today, month to date and trailing 30 days are calendar
	// windows (they include any history before day 1); since day 1 starts
	// at the start date.
	Today       FinancePeriod `json:"today"`
	MonthToDate FinancePeriod `json:"monthToDate"`
	Trailing30  FinancePeriod `json:"trailing30"`
	SinceStart  FinancePeriod `json:"sinceStart"`
	Claims      ClaimsSummary `json:"claims"`
	// CashHistory is actual end-of-day cash for up to the last 30 days,
	// oldest first; today's entry is the current balance. The takeover
	// capital (Opening.AdjustmentCents) appears once, on the takeover day.
	CashHistory  []DayAmount    `json:"cashHistory"`
	NextPayroll  Obligation     `json:"nextPayroll"`  // next payday and the payroll accrued so far
	NextPayables Obligation     `json:"nextPayables"` // next 1st of the month and the payables accrued so far
	Accounts     []Account      `json:"accounts"`     // trial balance
	BalanceSheet BalanceSheet   `json:"balanceSheet"`
	Opening      OpeningState   `json:"opening"`
	Financing    Financing      `json:"financing"`
	Journal      []JournalEntry `json:"journal"` // newest first (capped)
	JournalTotal int            `json:"journalTotal"`
}

// FinancePeriod is an income statement and cash flow for a date range.
type FinancePeriod struct {
	Label string `json:"label"`
	From  string `json:"from"` // dates, inclusive
	To    string `json:"to"`
	Days  int64  `json:"days"`
	// Revenue recognized: the expected net amount booked when each report
	// is finalized (allowed × expected collection rate). Collection
	// variance: on claims settled in the period, expected minus actually
	// collected (positive when less was collected). Net revenue = revenue
	// recognized − collection variance; it exceeds revenue recognized when
	// settled claims paid more than expected.
	GrossRevenueCents       int64 `json:"grossRevenueCents"`
	CollectionVarianceCents int64 `json:"collectionVarianceCents"`
	NetRevenueCents         int64 `json:"netRevenueCents"`
	// Operating expenses by account.
	WagesCents     int64 `json:"wagesCents"`
	BenefitsCents  int64 `json:"benefitsCents"`
	FacilityCents  int64 `json:"facilityCents"`
	EquipmentCents int64 `json:"equipmentCents"`
	SuppliesCents  int64 `json:"suppliesCents"`
	ITCents        int64 `json:"itCents"`
	BillingCents   int64 `json:"billingCents"`
	// The same expenses by behavior: labor (wages + benefits), fixed
	// (monthly costs accrued daily) and variable (per-study supplies and IT,
	// billing fees).
	LaborCents            int64 `json:"laborCents"`
	FixedCents            int64 `json:"fixedCents"`
	VariableCents         int64 `json:"variableCents"`
	OperatingExpenseCents int64 `json:"operatingExpenseCents"`
	OperatingProfitCents  int64 `json:"operatingProfitCents"`
	// MarginBP is operating profit / net revenue in basis points; it is
	// undefined (MarginDefined false) when net revenue is not positive.
	MarginBP      int64 `json:"marginBP"`
	MarginDefined bool  `json:"marginDefined"`
	// Cash flow: collections are payments received; disbursements are
	// payroll and bills paid.
	CollectionsCents   int64 `json:"collectionsCents"`
	DisbursementsCents int64 `json:"disbursementsCents"`
	NetCashFlowCents   int64 `json:"netCashFlowCents"`
	ReportsFinal       int   `json:"reportsFinal"`
	// Denials over insurance claims adjudicated in the period (first
	// adjudication; self-pay excluded).
	Adjudicated  int             `json:"adjudicated"`
	Denied       int             `json:"denied"`
	DenialRateBP int64           `json:"denialRateBP"`
	ByModality   []ModalityMoney `json:"byModality"`
	// FixedByItem splits fixed costs by underlying item (rent, utilities,
	// insurance, each lease and service contract, IT), whatever account
	// they post to.
	FixedByItem []ItemMoney `json:"fixedByItem"`
}

// ItemMoney is one fixed-cost item's expense in a period.
type ItemMoney struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Account string `json:"account"`
	Cents   int64  `json:"cents"`
}

// BalanceSheet: assets = liabilities + equity. Equity is the opening
// capital (the adjustment at the start), the operating result of the
// history before day 1 (retained earnings) and the result since day 1.
type BalanceSheet struct {
	CashCents               int64 `json:"cashCents"`
	ReceivablesCents        int64 `json:"receivablesCents"`
	AssetsCents             int64 `json:"assetsCents"`
	PayablesCents           int64 `json:"payablesCents"`
	AccruedPayrollCents     int64 `json:"accruedPayrollCents"`
	LiabilitiesCents        int64 `json:"liabilitiesCents"`
	OpeningCapitalCents     int64 `json:"openingCapitalCents"`
	HistoryEarningsCents    int64 `json:"historyEarningsCents"`
	EarningsSinceStartCents int64 `json:"earningsSinceStartCents"`
	EquityCents             int64 `json:"equityCents"`
}

// OpeningState describes the practice when play began: the opening cash
// adjustment and what the history left behind.
type OpeningState struct {
	Done        bool   `json:"done"`
	Time        string `json:"time"`
	HistoryDays int64  `json:"historyDays"`
	// Cash before the adjustment (what the history left), the scenario's
	// opening cash, and the adjustment between them: owner capital added
	// (> 0) or distributed (< 0).
	CashBeforeCents  int64 `json:"cashBeforeCents"`
	OpeningCashCents int64 `json:"openingCashCents"`
	AdjustmentCents  int64 `json:"adjustmentCents"`
	// Carried over from the history.
	ReceivablesCents    int64 `json:"receivablesCents"`
	PayablesCents       int64 `json:"payablesCents"`
	AccruedPayrollCents int64 `json:"accruedPayrollCents"`
	OpenClaims          int   `json:"openClaims"`
	Unread              int   `json:"unread"`
	Reading             int   `json:"reading"`
	BookedAppointments  int   `json:"bookedAppointments"` // from the start date on
	RecallList          int   `json:"recallList"`
	// The history's results (before day 1).
	HistoryNetRevenueCents int64 `json:"historyNetRevenueCents"`
	HistoryExpenseCents    int64 `json:"historyExpenseCents"`
	HistoryResultCents     int64 `json:"historyResultCents"`
}

// Financing tracks negative cash since the start. V0.1 has no credit line:
// a negative balance is an unfunded financing shortfall, shown and
// tracked, not free borrowing.
type Financing struct {
	ShortfallCents    int64  `json:"shortfallCents"` // max(0, −cash) now
	MinCashCents      int64  `json:"minCashCents"`   // lowest balance since the start
	MinCashTime       string `json:"minCashTime,omitempty"`
	DaysBelowZero     int64  `json:"daysBelowZero"` // days since day 1 that ended with cash < 0
	CreditLineModeled bool   `json:"creditLineModeled"`
}

// ModalityMoney is revenue and direct cost for one modality. Direct cost
// is supplies, per-study IT, billing fees and the modality's own leases
// and service contracts; shared labor and facility costs are not split.
type ModalityMoney struct {
	Modality        string `json:"modality"`
	ReportsFinal    int    `json:"reportsFinal"`
	NetRevenueCents int64  `json:"netRevenueCents"`
	DirectCostCents int64  `json:"directCostCents"`
}

// ClaimsSummary counts claims by state and ages the open ones.
type ClaimsSummary struct {
	Total      int           `json:"total"`
	Open       int           `json:"open"`
	Pending    int           `json:"pending"`
	Submitted  int           `json:"submitted"`
	Denied     int           `json:"denied"` // denied or resubmitted, still open
	Paid       int           `json:"paid"`
	ShortPaid  int           `json:"shortPaid"`
	WrittenOff int           `json:"writtenOff"`
	DaysInAR   int64         `json:"daysInAR"`   // A/R ÷ average daily net revenue over the trailing 30 days; -1 if undefined
	Aging      []AgingBucket `json:"aging"`      // by days since report final
	OpenClaims []ClaimItem   `json:"openClaims"` // oldest first (capped)
}

// AgingBucket is open A/R in an age range.
type AgingBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
	Cents int64  `json:"cents"`
}

// ClaimItem is one claim as the practice sees it: never its future outcome.
type ClaimItem struct {
	Claim         string `json:"claim"`
	Study         string `json:"study"`
	Patient       string `json:"patient"`
	Payer         string `json:"payer"`
	Modality      string `json:"modality"`
	ExamName      string `json:"examName"`
	State         string `json:"state"`
	Final         string `json:"final"`               // report final (revenue recognized)
	Submitted     string `json:"submitted,omitempty"` // first submission
	AgeDays       int64  `json:"ageDays"`             // since report final
	ExpectedCents int64  `json:"expectedCents"`
	PaidCents     int64  `json:"paidCents"`
	BalanceCents  int64  `json:"balanceCents"`
}

// Obligation is a known upcoming payment.
type Obligation struct {
	Date  string `json:"date"`
	Cents int64  `json:"cents"`
}

// DayAmount is one day's figure.
type DayAmount struct {
	Date  string `json:"date"`
	Cents int64  `json:"cents"`
}

// Account is one trial-balance line, in its normal-balance sign (assets
// and expenses debit-positive; liabilities, equity and revenue
// credit-positive; collection variance debit-positive).
type Account struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	BalanceCents int64  `json:"balanceCents"`
}

// JournalEntry is one posted double-entry transaction.
type JournalEntry struct {
	ID    string        `json:"id"`
	Time  string        `json:"time"`
	Memo  string        `json:"memo"`
	Ref   string        `json:"ref,omitempty"` // the operational record (study, claim, employee)
	Lines []JournalLine `json:"lines"`
}

// JournalLine is one debit or credit.
type JournalLine struct {
	Account     string `json:"account"`
	Modality    string `json:"modality,omitempty"`
	Item        string `json:"item,omitempty"` // fixed-cost item, e.g. "Utilities"
	DebitCents  int64  `json:"debitCents,omitempty"`
	CreditCents int64  `json:"creditCents,omitempty"`
}

// DaySummary is one day's record. BeforeStart marks history from before
// day 1 (the practice's past, never scored). Days are calendar days,
// midnight to midnight; the takeover day includes its activity before the
// takeover (Takeover gives the time). Complete is false for today (figures
// so far, as of AsOf). Waits are arrival to exam start for patients imaged
// that day; TAT is over reports finalized that day. CashEndCents is actual
// end-of-day cash; ProFormaCashEndCents adds the takeover capital to days
// before it (labelled pro forma). Ops and Money hold the full metrics; the
// flat fields are the headline figures.
type DaySummary struct {
	Date                 string        `json:"date"`
	Weekday              string        `json:"weekday"`
	BeforeStart          bool          `json:"beforeStart"`
	Complete             bool          `json:"complete"`
	AsOf                 string        `json:"asOf"`               // practice-local ISO time the figures run to
	Takeover             string        `json:"takeover,omitempty"` // takeover time, on the takeover day
	Orders               int           `json:"orders"`
	Booked               int           `json:"booked"`
	Lost                 int           `json:"lost"`
	Cancelled            int           `json:"cancelled"`
	WalkIns              int           `json:"walkIns"`
	Arrived              int           `json:"arrived"`
	NoShows              int           `json:"noShows"`
	Exams                int           `json:"exams"`
	ByModality           []ModalityDay `json:"byModality"`
	WaitMedianMin        int64         `json:"waitMedianMin"`
	ReportsFinal         int           `json:"reportsFinal"`
	TATMedianMin         int64         `json:"tatMedianMin"`
	UnreadAtClose        int           `json:"unreadAtClose"`
	OvertimeMin          int64         `json:"overtimeMin"`
	NetRevenueCents      int64         `json:"netRevenueCents"`
	ExpenseCents         int64         `json:"expenseCents"`
	OperatingProfitCents int64         `json:"operatingProfitCents"`
	CollectionsCents     int64         `json:"collectionsCents"`
	DisbursementsCents   int64         `json:"disbursementsCents"`
	CashEndCents         int64         `json:"cashEndCents"`
	// ProFormaCashEndCents is cash end adjusted as if the takeover capital
	// had been in place throughout (pro forma, not actual).
	ProFormaCashEndCents int64 `json:"proFormaCashEndCents"`
	// End-of-day balances (or now, for today): what is still owed and
	// outstanding.
	ARCents             int64         `json:"arCents"`
	APCents             int64         `json:"apCents"`
	AccruedPayrollCents int64         `json:"accruedPayrollCents"`
	Ops                 PeriodMetrics `json:"ops"`
	Money               FinancePeriod `json:"money"`
}

// ModalityDay is one modality's exams and room time for a day. Utilization
// is busy (patient in the room) over open hours and over staffed hours,
// in basis points.
type ModalityDay struct {
	Modality      string `json:"modality"`
	Exams         int    `json:"exams"`
	BusyMin       int64  `json:"busyMin"`
	OpenMin       int64  `json:"openMin"`
	StaffedMin    int64  `json:"staffedMin"`
	UtilOpenBP    int64  `json:"utilOpenBP"`
	UtilStaffedBP int64  `json:"utilStaffedBP"`
}

// Ratio is a rate kept as numerator and denominator, so an empty
// denominator reads as undefined ("—"), never as 0%.
type Ratio struct {
	Num int64 `json:"num"`
	Den int64 `json:"den"`
}

// Defined reports whether the denominator is positive.
func (r Ratio) Defined() bool { return r.Den > 0 }

// BP is the ratio in basis points (0 when undefined).
func (r Ratio) BP() int64 {
	if r.Den <= 0 {
		return 0
	}
	return r.Num * 10_000 / r.Den
}

// Dist summarizes durations in seconds. Percentiles are nearest-rank
// (the smallest value with at least p% of the sample at or below it); all
// zero when Count is 0.
type Dist struct {
	Count  int   `json:"count"`
	P50Sec int64 `json:"p50Sec"`
	P90Sec int64 `json:"p90Sec"`
	MaxSec int64 `json:"maxSec"`
}

// PeriodMetrics are the operational metrics of a period [From, To]
// (practice-local ISO times): a calendar day, today so far, or the
// evaluation window. Time-based figures are interval measurements in
// seconds (SIMULATION_DESIGN §11).
type PeriodMetrics struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Complete bool   `json:"complete"` // the period has ended; false: so far

	Demand          DemandMetrics    `json:"demand"`
	Appointments    ApptMetrics      `json:"appointments"`
	WalkIns         WalkInMetrics    `json:"walkIns"`
	Waits           WaitMetrics      `json:"waits"`
	Exams           int              `json:"exams"`           // exams completed
	AfterHoursExams int              `json:"afterHoursExams"` // started after scheduled close
	Rooms           []RoomMetrics    `json:"rooms"`
	Staff           []StaffMetrics   `json:"staff"`
	Reporting       ReportingMetrics `json:"reporting"`
	Workforce       WorkforceMetrics `json:"workforce"`
	Alerts          AlertCounts      `json:"alerts"`
	Actions         ActionCounts     `json:"actions"`
}

// ActionCounts are the actions attempted in a period.
type ActionCounts struct {
	Attempted int `json:"attempted"`
	Accepted  int `json:"accepted"` // incl. no-ops (setting the current policy)
	Rejected  int `json:"rejected"`
}

// DemandMetrics: orders received and what booking did with them. LostRate
// is lost / received.
type DemandMetrics struct {
	Received   int                     `json:"received"`
	Booked     int                     `json:"booked"` // incl. recall releases and rebookings
	Recall     int                     `json:"recall"`
	Lost       int                     `json:"lost"`
	Cancelled  int                     `json:"cancelled"` // appointments cancelled by patients
	LostRate   Ratio                   `json:"lostRate"`
	ByModality []ModalityDemandMetrics `json:"byModality"`
	// Booking events of walk-in-originated orders (offers, rebookings,
	// cancellations), kept out of the referral counters above (step 18.1).
	WalkInOrigin WalkInOriginDemand `json:"walkInOrigin"`
}

// WalkInOriginDemand counts booking events of walk-in-originated orders.
type WalkInOriginDemand struct {
	Booked    int `json:"booked"`
	Lost      int `json:"lost"`
	Cancelled int `json:"cancelled"`
}

// ModalityDemandMetrics is one modality's demand and output.
type ModalityDemandMetrics struct {
	Modality string `json:"modality"`
	Received int    `json:"received"`
	Lost     int    `json:"lost"`
	Exams    int    `json:"exams"`
}

// ApptMetrics covers the appointments whose slot starts in the period.
// Scheduled counts every appointment ever booked into those slots;
// NoShows includes Missed (provisional until accounting close). Rates:
// no-shows / (scheduled − cancelled); cancelled / scheduled. Fill is
// booked slot time over bookable slot time (slots a technologist, and for
// contrast an on-site radiologist, was rostered for).
type ApptMetrics struct {
	Scheduled   int   `json:"scheduled"`
	Cancelled   int   `json:"cancelled"`
	Arrived     int   `json:"arrived"`
	NoShows     int   `json:"noShows"`
	Missed      int   `json:"missed"` // of NoShows, not yet final
	Rescheduled int   `json:"rescheduled"`
	NoShowRate  Ratio `json:"noShowRate"`
	CancelRate  Ratio `json:"cancelRate"`
	BookableSec int64 `json:"bookableSec"`
	BookedSec   int64 `json:"bookedSec"`
	Fill        Ratio `json:"fill"`
}

// WalkInMetrics keeps four concepts apart (step 18.1): arrival, admission
// on the visit the walk-in came on, conversion to a scheduled appointment,
// and the exam. The activity counts are events in the period (an exam may
// belong to a walk-in who arrived earlier); Cohort follows the walk-ins who
// arrived in the period to their outcome as of its end. Acceptance =
// cohort imaged on the same visit / arrived; abandonment = cohort left /
// arrived.
type WalkInMetrics struct {
	Arrived  int `json:"arrived"`
	Admitted int `json:"admitted"` // taken in on the visit they came on
	// Offered an appointment instead: Declined by the walk-in policy, or
	// Unservable today; the offer booked or failed (order lost).
	Offered     int `json:"offered"`
	Declined    int `json:"declined"`
	Unservable  int `json:"unservable"`
	OfferBooked int `json:"offerBooked"`
	OfferFailed int `json:"offerFailed"`
	Left        int `json:"left"` // left without an exam
	// Exams of walk-in orders started in the period: on the walk-in visit
	// itself, and on a later scheduled visit.
	SameVisitExams int          `json:"sameVisitExams"`
	ReturnExams    int          `json:"returnExams"`
	Cohort         WalkInCohort `json:"cohort"`
	Acceptance     Ratio        `json:"acceptance"`
	Abandonment    Ratio        `json:"abandonment"`
}

// WalkInCohort classifies each walk-in who arrived in the period once, as
// of the period end: Arrived = ImagedSameVisit + Converted + Left + Lost +
// OnSite.
type WalkInCohort struct {
	Arrived         int `json:"arrived"`
	ImagedSameVisit int `json:"imagedSameVisit"`
	// Converted to a scheduled appointment (on arrival, or rescheduled
	// after admission); ConvertedImaged, of them, examined by the end.
	Converted       int `json:"converted"`
	ConvertedImaged int `json:"convertedImaged"`
	Left            int `json:"left"`   // left without an exam
	Lost            int `json:"lost"`   // no appointment acceptable: order lost
	OnSite          int `json:"onSite"` // still on site, unresolved
}

// WaitMetrics are over exams started in the period: arrival to
// registration start, arrival to exam start, and (scheduled patients)
// exam start past the appointment time (0 when on time or early).
type WaitMetrics struct {
	Registration    Dist `json:"registration"`
	ToExam          Dist `json:"toExam"`
	PastAppointment Dist `json:"pastAppointment"`
	Over30          int  `json:"over30"` // arrival to exam start over 30 minutes
}

// RoomMetrics is one room's time. Open: the practice's scheduled hours.
// Staffed: open time with a qualified technologist on duty (a shared
// technologist counts fully for each room they cover). Busy: a patient in
// the room during open hours; after-hours busy is reported separately.
// UtilOpen = busy / open, UtilStaffed = busy / staffed, Coverage = staffed
// / open; none can exceed 100%.
type RoomMetrics struct {
	Room              string `json:"room"`
	Modality          string `json:"modality"`
	OpenSec           int64  `json:"openSec"`
	StaffedSec        int64  `json:"staffedSec"`
	BusySec           int64  `json:"busySec"`
	AfterHoursBusySec int64  `json:"afterHoursBusySec"`
	Exams             int    `json:"exams"`
	UtilOpen          Ratio  `json:"utilOpen"`
	UtilStaffed       Ratio  `json:"utilStaffed"`
	Coverage          Ratio  `json:"coverage"`
}

// StaffMetrics is one employee's working time. On duty is paid working
// time (breaks excluded, overtime included); busy is registration, exam
// and documentation (technologists, desk) or reading (radiologists); Util
// = busy / on duty. Radiologists: site-reading block time, reads inside and
// outside blocks, and ReadUtil = read inside blocks / block time.
type StaffMetrics struct {
	Name                string `json:"name"`
	Role                string `json:"role"`
	OnDutySec           int64  `json:"onDutySec"`
	BusySec             int64  `json:"busySec"`
	BreakSec            int64  `json:"breakSec"`
	OvertimeSec         int64  `json:"overtimeSec"`         // on duty past the rostered shift end
	ApprovedOvertimeSec int64  `json:"approvedOvertimeSec"` // of it, approved in advance
	Util                Ratio  `json:"util"`
	BlockSec            int64  `json:"blockSec,omitempty"`
	ReadInBlockSec      int64  `json:"readInBlockSec,omitempty"`
	ReadOutsideSec      int64  `json:"readOutsideSec,omitempty"`
	ReadUtil            Ratio  `json:"readUtil"`
}

// ReportingMetrics: TAT over the studies acquired in the period, with
// unfinished ones counted at their age at the period end (censored, §11);
// TAT of the reports finalized in the period (Inherited: of studies
// acquired before the period); target breaches; worklist at both ends.
type ReportingMetrics struct {
	Acquired     int               `json:"acquired"`
	TAT          Dist              `json:"tat"`
	Unfinished   int               `json:"unfinished"`
	OverTarget   int               `json:"overTarget"` // of Acquired, TAT (censored) past target
	Final        int               `json:"final"`
	FinalTAT     Dist              `json:"finalTAT"`
	Inherited    int               `json:"inherited"`
	Breaches     int               `json:"breaches"` // targets missed (urgent/soon when passed, routine at final)
	ByPriority   []PriorityMetrics `json:"byPriority"`
	BacklogStart int               `json:"backlogStart"` // unread + reading
	BacklogEnd   int               `json:"backlogEnd"`
}

// PriorityMetrics is the censored TAT of one priority's studies.
type PriorityMetrics struct {
	Priority   string `json:"priority"`
	Acquired   int    `json:"acquired"`
	TAT        Dist   `json:"tat"`
	OverTarget int    `json:"overTarget"`
	TargetSec  int64  `json:"targetSec"`
}

// WorkforceMetrics: overtime (all of it required hold-over in V0.1),
// staff held over, breaks skipped, patients rescheduled for lack of safe
// coverage, and patient-seconds spent waiting to register while no desk
// staff was on duty.
type WorkforceMetrics struct {
	OvertimeSec       int64 `json:"overtimeSec"`
	OvertimeCostCents int64 `json:"overtimeCostCents"`
	// Of the overtime: approved in advance (extend_shift); the rest was
	// required hold-over.
	ApprovedOvertimeSec       int64 `json:"approvedOvertimeSec"`
	ApprovedOvertimeCostCents int64 `json:"approvedOvertimeCostCents"`
	HeldOver                  int   `json:"heldOver"`
	SkippedBreaks             int   `json:"skippedBreaks"`
	NoCoverage                int   `json:"noCoverage"`
	DeskSelfPatientSec        int64 `json:"deskSelfPatientSec"`
}

// AlertCounts are the alerts raised in a period, by severity, with the
// first few texts.
type AlertCounts struct {
	Warn  int      `json:"warn"`
	Alert int      `json:"alert"`
	Top   []string `json:"top"`
}

// AlertItem is one alert: a condition the practice can see, raised once
// per episode (escalating at most once), resolved when the condition has
// cleared. Why explains it from observed facts; Screen is the TUI key
// that shows more.
type AlertItem struct {
	Key      string   `json:"key"`
	Severity Severity `json:"severity"`
	Since    string   `json:"since"`              // practice-local ISO time raised
	Resolved string   `json:"resolved,omitempty"` // when it cleared (or expired)
	Expired  bool     `json:"expired,omitempty"`  // ended with its day, not cleared
	Text     string   `json:"text"`
	Why      string   `json:"why"`
	Screen   string   `json:"screen"`
}

// Alerts are the active alerts (most severe first, then oldest) and the
// latest resolved ones (newest first, capped).
type Alerts struct {
	Active      []AlertItem `json:"active"`
	Resolved    []AlertItem `json:"resolved"`
	RaisedToday int         `json:"raisedToday"`
}

// Evaluation is the evaluation window: from the exact takeover time to the
// configured endpoint (or now, while it runs), measured against an
// immutable baseline captured at takeover. It is a measurement, not a
// score: weights and thresholds come later.
type Evaluation struct {
	Started  bool          `json:"started"`
	Baseline Baseline      `json:"baseline"`
	End      string        `json:"end,omitempty"` // configured endpoint; empty: open-ended
	Ended    bool          `json:"ended"`
	AsOf     string        `json:"asOf"`
	Ops      PeriodMetrics `json:"ops"`
	Money    WindowMoney   `json:"money"`
}

// Baseline is the state at the takeover instant, after everything at that
// instant (the opening adjustment included). Digest identifies it.
type Baseline struct {
	Time   string `json:"time"`
	Digest string `json:"digest"`

	CashCents           int64       `json:"cashCents"`
	ReceivablesCents    int64       `json:"receivablesCents"`
	PayablesCents       int64       `json:"payablesCents"`
	AccruedPayrollCents int64       `json:"accruedPayrollCents"`
	EquityCents         int64       `json:"equityCents"` // all equity: opening capital and retained results
	Accounts            []Account   `json:"accounts"`
	Claims              []ClaimLoad `json:"claims"` // open claims by state
	OpenClaims          int         `json:"openClaims"`
	// Time-based accruals posted at the takeover day's midnight that cover
	// the rest of that day: the window bears the share after takeover.
	AllocatedAccrualCents int64 `json:"allocatedAccrualCents"`

	Unreported         int      `json:"unreported"` // studies acquired, not final
	Unread             int      `json:"unread"`
	Reading            int      `json:"reading"`
	OldestUnreportedS  int64    `json:"oldestUnreportedSec"`
	BookedAppointments int      `json:"bookedAppointments"` // from the takeover on
	RecallList         int      `json:"recallList"`
	OnSite             int      `json:"onSite"`
	StaffOnDuty        []string `json:"staffOnDuty"`
	RoomsStaffed       int      `json:"roomsStaffed"`
	LedgerEntries      int      `json:"ledgerEntries"`
	LogSeq             uint64   `json:"logSeq"`
}

// ClaimLoad is the open claims in one state.
type ClaimLoad struct {
	State string `json:"state"`
	Count int    `json:"count"`
	Cents int64  `json:"cents"`
}

// WindowMoney is the evaluation window's money. Ledger is every journal
// entry posted after the baseline, as posted. Operating performance then
// removes what is not the window's: collection variance on claims whose
// revenue was recognized before takeover (a true-up of the history's
// revenue) and adds the share of the takeover day's midnight accruals that
// covers time after takeover. Capital transfers are never operating
// performance. The cash bridge: operating profit − inherited variance +
// allocated accruals (already in the baseline's liabilities) − ΔA/R +
// ΔA/P + Δpayroll owed = operating cash flow (collections −
// disbursements); + capital transfers = change in cash.
type WindowMoney struct {
	Ledger FinancePeriod `json:"ledger"`

	InheritedVarianceCents int64 `json:"inheritedVarianceCents"`
	AllocatedAccrualCents  int64 `json:"allocatedAccrualCents"`
	// The fields below are baseline-adjusted (an evaluation convention);
	// the ordinary accrual P&L, with the actual collection variance, is
	// Ledger (and the calendar statements).
	RevenueCents          int64 `json:"revenueCents"` // recognized in the window
	NetRevenueCents       int64 `json:"netRevenueCents"`
	OperatingExpenseCents int64 `json:"operatingExpenseCents"`
	OperatingProfitCents  int64 `json:"operatingProfitCents"`
	Margin                Ratio `json:"margin"` // operating profit / net revenue (undefined unless revenue > 0)

	CollectionsCents          int64 `json:"collectionsCents"`
	InheritedCollectionsCents int64 `json:"inheritedCollectionsCents"` // on claims recognized before takeover
	DisbursementsCents        int64 `json:"disbursementsCents"`
	InheritedPaidCents        int64 `json:"inheritedPaidCents"` // of disbursements, settling baseline payables and payroll
	OperatingCashFlowCents    int64 `json:"operatingCashFlowCents"`
	CapitalTransfersCents     int64 `json:"capitalTransfersCents"`
	CashChangeCents           int64 `json:"cashChangeCents"`
	ARChangeCents             int64 `json:"arChangeCents"`
	APChangeCents             int64 `json:"apChangeCents"`
	PayrollChangeCents        int64 `json:"payrollChangeCents"`

	CashEndCents   int64 `json:"cashEndCents"`
	MinCashCents   int64 `json:"minCashCents"`
	ShortfallCents int64 `json:"shortfallCents"` // max(0, −lowest cash in the window)
	Reconciled     bool  `json:"reconciled"`     // the cash bridge holds to the cent
}

// Action types (SIMULATION_DESIGN §9.5). Every frontend and agent submits
// these through the same Apply; validation lives in the simulation.
const (
	ActExtendShift       = "extend_shift"
	ActSetWalkInPolicy   = "set_walkin_policy"
	ActSetWorklistPolicy = "set_worklist_policy"
)

// Walk-in policies.
const (
	WalkInAcceptAlways  = "accept_always"  // wait for the next free technologist
	WalkInAcceptIfGap   = "accept_if_gap"  // only into a gap before the next due scheduled patient
	WalkInAcceptUntil   = "accept_until"   // accept always until the cutoff, then decline
	WalkInScheduleLater = "schedule_later" // decline: offer an appointment instead
)

// Worklist policies.
const (
	WorklistUrgentThenFIFO = "urgent_then_fifo"
	WorklistFIFO           = "fifo"
	WorklistOldestFirst    = "oldest_first"
	WorklistShortestFirst  = "shortest_first" // by the published expected read time
)

// Action is one decision. Fields by type: extend_shift — Employee (ID),
// Minutes and optionally Date; set_walkin_policy — Policy and, for accept_until, Cutoff
// ("HH:MM"); set_worklist_policy — Policy.
type Action struct {
	Type     string `json:"type"`
	Employee string `json:"employee,omitempty"`
	Minutes  int64  `json:"minutes,omitempty"`
	// Date is the day an extension applies to ("YYYY-MM-DD"): today (the
	// default) or a later day within the booking horizon. Approved ahead,
	// its slots open for booking at once.
	Date   string `json:"date,omitempty"`
	Policy string `json:"policy,omitempty"`
	Cutoff string `json:"cutoff,omitempty"`
}

// Result codes other than rejections.
const (
	ActionAccepted  = "accepted"
	ActionUnchanged = "unchanged" // valid, but already in force: nothing changed
)

// ActionResult is the outcome of one action. ID numbers every attempted
// action in order ("X00001"); Batch numbers decisions. Time is when it
// took effect (or was refused): the clock at the batch. A rejected action
// changed nothing; Code says why ("exceeds_max_shift", …).
type ActionResult struct {
	ID      string `json:"id"`
	Batch   int64  `json:"batch"`
	Time    string `json:"time"`
	Action  Action `json:"action"`
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ActionLog counts every action attempted and lists the latest.
type ActionLog struct {
	Attempted int            `json:"attempted"`
	Accepted  int            `json:"accepted"`
	Rejected  int            `json:"rejected"`
	Recent    []ActionResult `json:"recent"` // newest first (capped)
}

// RosterEntry is one employee's shift on a day, with any approved
// extension: Shift is the shift as now scheduled, RosterEnd the rostered
// end, CapAt the latest end the shift cap allows.
type RosterEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Shift       string `json:"shift"`
	RosterEnd   string `json:"rosterEnd"`
	ApprovedMin int64  `json:"approvedMin"`
	CapAt       string `json:"capAt"`
	HourlyCents int64  `json:"hourlyCents,omitempty"`
}

// Extension is an approved shift extension, today or ahead.
type Extension struct {
	Date     string `json:"date"`
	Employee string `json:"employee"`
	Name     string `json:"name"`
	Minutes  int64  `json:"minutes"`
	Until    string `json:"until"` // the extended end
}

// Policies are the operating policies in force.
type Policies struct {
	WalkIn       string `json:"walkIn"`
	WalkInCutoff string `json:"walkInCutoff,omitempty"` // accept_until: "HH:MM"
	Worklist     string `json:"worklist"`
}
