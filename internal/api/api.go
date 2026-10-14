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
}

// Room states (SIMULATION_DESIGN §2.5).
const (
	RoomClosed    = "closed"
	RoomUnstaffed = "unstaffed"
	RoomIdle      = "idle"
)

// Room is one room's observable status.
type Room struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Modality string   `json:"modality"`
	State    string   `json:"state"`
	StaffBy  []string `json:"staffedBy"` // employee IDs
	// Shared is true when a technologist staffing this room also staffs
	// another one.
	Shared bool `json:"shared"`
}

// Employee statuses.
const (
	StaffOff         = "off"
	StaffOnDuty      = "on_duty"
	StaffOnBreak     = "on_break"
	StaffReading     = "reading"      // radiologist in a site reading block
	StaffOtherDuties = "other_duties" // radiologist on shift, outside site blocks
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
	Unseen        int `json:"unseen"`        // still waiting at closing time, sent home
}

// WaitingPatient is a patient on site who has not been imaged. In V0.1
// step 7 nobody is registered yet: waiting means arrived.
type WaitingPatient struct {
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
	Waiting          []WaitingPatient `json:"waiting"` // in arrival order
}

// Patient status keys for lists.
const (
	PatientBooked      = "booked"
	PatientCancelled   = "cancelled"
	PatientWaiting     = "waiting"
	PatientMissed      = "missed"
	PatientNoShow      = "no_show"
	PatientRescheduled = "rescheduled"
	PatientLeft        = "left"
	PatientUnseen      = "unseen"
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
