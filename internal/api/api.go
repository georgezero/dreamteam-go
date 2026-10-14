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
	Arrived       int `json:"arrived"`       // scheduled patients who arrived
	Early         int `json:"early"`         // … more than 5 min early
	Late          int `json:"late"`          // … more than 5 min late
	AfterGrace    int `json:"afterGrace"`    // … after being marked missed (fitted in)
	WalkIns       int `json:"walkIns"`       // walk-ins who came in
	WalkInsBooked int `json:"walkInsBooked"` // … sent away with an appointment instead
	Missed        int `json:"missed"`        // appointments marked missed at start + grace
	NoShows       int `json:"noShows"`       // missed appointments finalized at accounting close
	Rescheduled   int `json:"rescheduled"`   // too late for the late policy, or after closing
	Left          int `json:"left"`          // walk-ins who left without an exam

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
