// Package simtime provides integer simulated time and a practice-local
// calendar. Time is seconds since the scenario epoch (00:00 on the epoch
// date). There are no time zones and no daylight saving time.
//
// Nothing in this package reads the wall clock.
package simtime

import (
	"fmt"
	"strconv"
	"strings"
)

// Time is seconds since the scenario epoch.
type Time int64

// Duration is a span of simulated seconds.
type Duration int64

const (
	Second Duration = 1
	Minute          = 60 * Second
	Hour            = 60 * Minute
	Day             = 24 * Hour
)

// Add returns t+d.
func (t Time) Add(d Duration) Time { return t + Time(d) }

// Sub returns t-u.
func (t Time) Sub(u Time) Duration { return Duration(t - u) }

// Minutes returns d in whole minutes, truncated toward zero.
func (d Duration) Minutes() int64 { return int64(d / Minute) }

// String formats d compactly: "45s", "34m", "3h 12m", "2d 4h".
func (d Duration) String() string {
	neg := d < 0
	if neg {
		d = -d
	}
	var s string
	switch {
	case d < Minute:
		s = fmt.Sprintf("%ds", int64(d))
	case d < Hour:
		s = fmt.Sprintf("%dm", int64(d/Minute))
	case d < Day:
		h, m := int64(d/Hour), int64(d%Hour/Minute)
		if m == 0 {
			s = fmt.Sprintf("%dh", h)
		} else {
			s = fmt.Sprintf("%dh %02dm", h, m)
		}
	default:
		days, h := int64(d/Day), int64(d%Day/Hour)
		if h == 0 {
			s = fmt.Sprintf("%dd", days)
		} else {
			s = fmt.Sprintf("%dd %dh", days, h)
		}
	}
	if neg {
		return "-" + s
	}
	return s
}

// Weekday follows time.Weekday numbering: Sunday = 0.
type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var weekdayNames = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func (w Weekday) String() string {
	if w < 0 || w > 6 {
		return "Weekday(" + strconv.Itoa(int(w)) + ")"
	}
	return weekdayNames[w]
}

// ParseWeekday accepts three-letter English abbreviations ("Mon").
func ParseWeekday(s string) (Weekday, error) {
	for i, n := range weekdayNames {
		if strings.EqualFold(s, n) {
			return Weekday(i), nil
		}
	}
	return 0, fmt.Errorf("simtime: unknown weekday %q", s)
}

// Date is a proleptic Gregorian calendar date.
type Date struct {
	Year  int
	Month int // 1-12
	Day   int // 1-31
}

var monthNames = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// ParseDate parses "YYYY-MM-DD".
func ParseDate(s string) (Date, error) {
	var d Date
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return d, fmt.Errorf("simtime: date %q is not YYYY-MM-DD", s)
	}
	y, err1 := strconv.Atoi(s[0:4])
	m, err2 := strconv.Atoi(s[5:7])
	dd, err3 := strconv.Atoi(s[8:10])
	if err1 != nil || err2 != nil || err3 != nil {
		return d, fmt.Errorf("simtime: date %q is not YYYY-MM-DD", s)
	}
	d = Date{y, m, dd}
	if !d.Valid() {
		return Date{}, fmt.Errorf("simtime: invalid date %q", s)
	}
	return d, nil
}

// Valid reports whether d names a real calendar day.
func (d Date) Valid() bool {
	if d.Month < 1 || d.Month > 12 || d.Day < 1 {
		return false
	}
	return d.Day <= daysInMonth(d.Year, d.Month)
}

func daysInMonth(y, m int) int {
	switch m {
	case 2:
		if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// DaysInMonth returns the number of days in d's month.
func (d Date) DaysInMonth() int { return daysInMonth(d.Year, d.Month) }

// MonthName returns "Jan".."Dec".
func (d Date) MonthName() string { return monthNames[d.Month-1] }

// Ordinal returns days since 1970-01-01 (negative before).
// Algorithm: Howard Hinnant, "days_from_civil".
func (d Date) Ordinal() int64 {
	y := int64(d.Year)
	m := int64(d.Month)
	if m <= 2 {
		y--
	}
	era := floorDiv(y, 400)
	yoe := y - era*400
	mp := (m + 9) % 12
	doy := (153*mp+2)/5 + int64(d.Day) - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

// DateFromOrdinal inverts Ordinal. Algorithm: "civil_from_days".
func DateFromOrdinal(z int64) Date {
	z += 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	day := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if m > 12 {
		m -= 12
	}
	if m <= 2 {
		y++
	}
	return Date{int(y), int(m), int(day)}
}

// AddDays returns the date n days after d.
func (d Date) AddDays(n int64) Date { return DateFromOrdinal(d.Ordinal() + n) }

// DaysSince returns d - e in days.
func (d Date) DaysSince(e Date) int64 { return d.Ordinal() - e.Ordinal() }

// Weekday returns the day of the week. 1970-01-01 was a Thursday.
func (d Date) Weekday() Weekday {
	return Weekday(floorMod(d.Ordinal()+4, 7))
}

// Calendar maps simulated time onto practice-local dates.
type Calendar struct {
	epoch Date
	base  int64 // epoch ordinal
}

// NewCalendar returns a calendar whose Time 0 is 00:00 on epoch.
func NewCalendar(epoch Date) Calendar {
	return Calendar{epoch: epoch, base: epoch.Ordinal()}
}

// Epoch returns the date of Time 0.
func (c Calendar) Epoch() Date { return c.epoch }

// DayIndex returns the number of whole days from the epoch to t.
func (c Calendar) DayIndex(t Time) int64 { return floorDiv(int64(t), int64(Day)) }

// Date returns the calendar date containing t.
func (c Calendar) Date(t Time) Date { return DateFromOrdinal(c.base + c.DayIndex(t)) }

// Weekday returns the day of the week containing t.
func (c Calendar) Weekday(t Time) Weekday { return c.Date(t).Weekday() }

// StartOfDay returns 00:00 of day index i.
func (c Calendar) StartOfDay(i int64) Time { return Time(i * int64(Day)) }

// StartOfDate returns 00:00 on d.
func (c Calendar) StartOfDate(d Date) Time { return c.StartOfDay(d.Ordinal() - c.base) }

// ClockOf returns the time since midnight of t.
func ClockOf(t Time) Duration { return Duration(floorMod(int64(t), int64(Day))) }

// At returns the instant clock (since midnight) on day index i.
func (c Calendar) At(i int64, clock Duration) Time { return c.StartOfDay(i).Add(clock) }

// ParseClock parses "HH:MM" or "HH:MM:SS" (00:00 to 24:00) into a Duration
// since midnight.
func ParseClock(s string) (Duration, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("simtime: clock %q is not HH:MM", s)
	}
	var v [3]int
	for i, p := range parts {
		if len(p) != 2 {
			return 0, fmt.Errorf("simtime: clock %q is not HH:MM", s)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("simtime: clock %q is not HH:MM", s)
		}
		v[i] = n
	}
	h, m, sec := v[0], v[1], v[2]
	if m > 59 || sec > 59 || h > 24 || (h == 24 && (m != 0 || sec != 0)) {
		return 0, fmt.Errorf("simtime: clock %q out of range", s)
	}
	return Duration(h)*Hour + Duration(m)*Minute + Duration(sec), nil
}

// FormatClock formats a time of day as "HH:MM".
func FormatClock(d Duration) string {
	d = Duration(floorMod(int64(d), int64(Day)))
	return fmt.Sprintf("%02d:%02d", int64(d/Hour), int64(d%Hour/Minute))
}

// Clock formats t as "HH:MM".
func (c Calendar) Clock(t Time) string { return FormatClock(ClockOf(t)) }

// Short formats t as "Tue Oct 13 10:42".
func (c Calendar) Short(t Time) string {
	d := c.Date(t)
	return fmt.Sprintf("%s %s %d %s", d.Weekday(), d.MonthName(), d.Day, c.Clock(t))
}

// ISO formats t as "2026-10-13T10:42:05" (practice local, no zone).
func (c Calendar) ISO(t Time) string {
	cl := ClockOf(t)
	return fmt.Sprintf("%sT%02d:%02d:%02d", c.Date(t),
		int64(cl/Hour), int64(cl%Hour/Minute), int64(cl%Minute))
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 { return a - floorDiv(a, b)*b }
