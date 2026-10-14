package simtime

import (
	"testing"
	"time"
)

func TestOrdinalMatchesStdlib(t *testing.T) {
	// The standard library is used only as a test oracle.
	start := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 200*366; i += 7 {
		tm := start.AddDate(0, 0, i)
		d := Date{tm.Year(), int(tm.Month()), tm.Day()}
		want := tm.Unix() / 86400
		if got := d.Ordinal(); got != want {
			t.Fatalf("%v ordinal %d, want %d", d, got, want)
		}
		if back := DateFromOrdinal(want); back != d {
			t.Fatalf("round trip %v -> %v", d, back)
		}
		if got, want := d.Weekday(), Weekday(tm.Weekday()); got != want {
			t.Fatalf("%v weekday %v, want %v", d, got, want)
		}
	}
}

func TestKnownDates(t *testing.T) {
	cases := []struct {
		s  string
		wd Weekday
	}{
		{"2026-10-13", Tuesday},
		{"2026-04-16", Thursday}, // 180 days before the default start
		{"2000-02-29", Tuesday},
		{"1970-01-01", Thursday},
	}
	for _, c := range cases {
		d, err := ParseDate(c.s)
		if err != nil {
			t.Fatal(err)
		}
		if d.Weekday() != c.wd || d.String() != c.s {
			t.Errorf("%s: weekday %v string %s", c.s, d.Weekday(), d)
		}
	}
	start, _ := ParseDate("2026-10-13")
	if got := start.AddDays(-180).String(); got != "2026-04-16" {
		t.Errorf("start-180 = %s", got)
	}
}

func TestParseDateErrors(t *testing.T) {
	for _, s := range []string{"", "2026-1-13", "2026-13-01", "2026-02-29", "2026/10/13", "abcd-ef-gh"} {
		if _, err := ParseDate(s); err == nil {
			t.Errorf("ParseDate(%q) accepted", s)
		}
	}
}

func TestCalendar(t *testing.T) {
	epoch, _ := ParseDate("2026-10-12") // Monday
	c := NewCalendar(epoch)
	at := c.At(1, 10*Hour+42*Minute+5)
	if got := c.Short(at); got != "Tue Oct 13 10:42" {
		t.Errorf("Short = %q", got)
	}
	if got := c.ISO(at); got != "2026-10-13T10:42:05" {
		t.Errorf("ISO = %q", got)
	}
	if c.DayIndex(at) != 1 || c.Weekday(at) != Tuesday {
		t.Errorf("day %d weekday %v", c.DayIndex(at), c.Weekday(at))
	}
	if got := c.StartOfDate(Date{2026, 10, 13}); got != c.StartOfDay(1) {
		t.Errorf("StartOfDate = %d", got)
	}
	// Negative times fall on earlier days.
	if c.DayIndex(-1) != -1 || c.Clock(-1) != "23:59" {
		t.Errorf("negative time: day %d clock %s", c.DayIndex(-1), c.Clock(-1))
	}
}

func TestParseClock(t *testing.T) {
	good := map[string]Duration{
		"00:00": 0, "07:45": 7*Hour + 45*Minute, "20:00": 20 * Hour,
		"24:00": 24 * Hour, "10:42:05": 10*Hour + 42*Minute + 5,
	}
	for s, want := range good {
		got, err := ParseClock(s)
		if err != nil || got != want {
			t.Errorf("ParseClock(%q) = %d, %v", s, got, err)
		}
	}
	for _, s := range []string{"7:45", "24:01", "12:60", "aa:bb", "12", "12:00:00:00", "-1:00"} {
		if _, err := ParseClock(s); err == nil {
			t.Errorf("ParseClock(%q) accepted", s)
		}
	}
	if FormatClock(7*Hour+45*Minute) != "07:45" {
		t.Error("FormatClock")
	}
}

func TestDurationString(t *testing.T) {
	cases := map[Duration]string{
		45: "45s", 34 * Minute: "34m", 3*Hour + 12*Minute: "3h 12m",
		2 * Hour: "2h", 2*Day + 4*Hour: "2d 4h", -5 * Minute: "-5m",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("%d: %q want %q", d, got, want)
		}
	}
}

func TestWeekdayParse(t *testing.T) {
	for i := Sunday; i <= Saturday; i++ {
		got, err := ParseWeekday(i.String())
		if err != nil || got != i {
			t.Errorf("ParseWeekday(%s) = %v, %v", i, got, err)
		}
	}
	if _, err := ParseWeekday("Funday"); err == nil {
		t.Error("accepted Funday")
	}
}
