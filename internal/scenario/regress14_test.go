package scenario

import (
	"strings"
	"testing"
)

// Step 14 review finding 1: hour keys must be the canonical "00"–"23";
// an alias such as "+8" for "08" made compilation depend on map order.
func TestRegress14HourKeysCanonical(t *testing.T) {
	for _, walkIns := range []bool{false, true} {
		f, err := ResolveFile("mini")
		if err != nil {
			t.Fatal(err)
		}
		m := f.Demand.HourProfileBP
		if walkIns {
			m = f.WalkIns.HourProfileBP
		}
		v := m["08"]
		if v < 2 {
			t.Fatalf("mini has no 08:00 share (%d)", v)
		}
		m["08"], m["+8"] = v/2, v-v/2 // same sum: only the alias is wrong
		if _, err := Compile(f); err == nil || !strings.Contains(err.Error(), "+8") {
			t.Errorf("walk-ins %v: alias hour key accepted (%v)", walkIns, err)
		}
	}
}

// Step 14 review finding 5: decoding a resolved scenario rejects anything
// after the value.
func TestRegress14ScenarioDecodeRejectsTrailing(t *testing.T) {
	f, _ := ResolveFile("mini")
	raw, _ := Canonical(f)
	for _, tail := range []string{"}", "]", "x", "{}"} {
		if _, err := Decode(append(append([]byte{}, raw...), tail...)); err == nil {
			t.Errorf("trailing %q accepted", tail)
		}
	}
	if _, err := Decode(append(append([]byte{}, raw...), " \n"...)); err != nil {
		t.Errorf("trailing whitespace rejected: %v", err)
	}
}
