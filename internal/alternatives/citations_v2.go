package alternatives

import (
	"encoding/json"
	"math/big"
	"regexp"
	"slices"
	"strings"
)

type citationValueV2 struct {
	Text   string
	Number *int64
	Unit   string
}

func citationItemV2(it ViewItem) citationValueV2 {
	k := viewKinds[it.Kind]
	unit := k.unit
	if strings.Contains(it.Kind, "minutes") {
		unit = "minutes"
	}
	if strings.HasPrefix(it.Kind, "slots_closed") {
		unit = "slots"
	}
	if k.show == showMinutes {
		unit = "minutes"
	}
	n := it.Number
	if it.Of != nil {
		n = nil
	}
	return citationValueV2{viewValue(it, k), n, unit}
}

var countTextV2 = regexp.MustCompile(`^-?(?:[0-9]+|[1-9][0-9]{0,2}(?:,[0-9]{3})+)$`)

func citationEqualV2(raw json.RawMessage, want citationValueV2) bool {
	var typed struct {
		Value json.RawMessage `json:"value"`
		Unit  string          `json:"unit"`
	}
	if len(raw) > 0 && raw[0] == '{' {
		if err := strict(raw, &typed); err != nil || want.Number == nil {
			return false
		}
		var n json.Number
		if json.Unmarshal(typed.Value, &n) != nil {
			return false
		}
		r, ok := new(big.Rat).SetString(string(n))
		if !ok {
			return false
		}
		if typed.Unit == "dollars" && want.Unit == UnitCents {
			r.Mul(r, big.NewRat(100, 1))
		} else if typed.Unit != want.Unit {
			return false
		}
		return r.Cmp(new(big.Rat).SetInt64(*want.Number)) == 0
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if s == want.Text {
			return true
		}
		if want.Number == nil || (want.Unit != "count" && want.Unit != "slots" && want.Unit != "bookings" && want.Unit != "minutes" && want.Unit != "days") || !countTextV2.MatchString(s) {
			return false
		}
		raw = []byte(strings.ReplaceAll(s, ",", ""))
	}
	if want.Number == nil || want.Unit == UnitCents || want.Unit == "dollars" {
		return false
	}
	var n json.Number
	if json.Unmarshal(raw, &n) != nil {
		return false
	}
	r, ok := new(big.Rat).SetString(string(n))
	if !ok {
		return false
	}
	return r.Cmp(new(big.Rat).SetInt64(*want.Number)) == 0
}

func verifyCitationsV2(v *ViewV2, inspected []string, cs []Citation, chosen string) (bool, []string) {
	var fails []string
	shown := map[string]bool{}
	values := map[string]map[string]citationValueV2{} // alt -> key -> shown value
	for _, x := range v.Alternatives {
		if !slices.Contains(inspected, x.ID) {
			continue
		}
		values[x.ID] = map[string]citationValueV2{}
		for key, q := range map[string]QuantityV2{"slots_closed": x.Capacity.Closed, "slots_opened": x.Capacity.Opened, "bookings_displaced": x.Capacity.Displaced, "replacement_capacity": x.Capacity.Replacements} {
			if q.Number != nil {
				shown[key] = true
				values[x.ID][key] = citationValueV2{Text: count(*q.Number), Number: q.Number, Unit: q.Unit}
			}
		}
		for _, list := range [][]ViewItem{x.Facts, x.Estimates, x.Obligations} {
			for _, it := range list {
				shown[it.Key] = true
				values[x.ID][it.Key] = citationItemV2(it)
			}
		}
	}
	if len(cs) > MaxCitations || len(cs) >= len(shown) {
		fails = append(fails, "too many entries: at most three, and fewer than the items shown")
	}
	// Every numeric item shown for every accepted alternative: citing all
	// of them is copying the table.
	var rows []string
	for k := range shown {
		all := true
		for _, x := range v.Alternatives {
			if !x.Accepted || values[x.ID] == nil {
				continue
			}
			if _, ok := values[x.ID][k]; !ok {
				all = false
			}
		}
		if all {
			rows = append(rows, k)
		}
	}
	cited := map[string]bool{}
	for _, c := range cs {
		cited[c.Item] = true
	}
	if len(rows) > 0 && !slices.ContainsFunc(rows, func(k string) bool { return !cited[k] }) {
		fails = append(fails, "repeats every row shown for every alternative")
	}
	differ := false
	for _, c := range cs {
		if len(c.Values) < 2 {
			fails = append(fails, c.Item+": names fewer than two alternatives")
			continue
		}
		other := false
		var seen []string
		ids := make([]string, 0, len(c.Values))
		for id := range c.Values {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		for _, id := range ids {
			if id != chosen {
				other = true
			}
			want, ok := values[id][c.Item]
			if !ok {
				fails = append(fails, c.Item+": not shown for "+id)
				continue
			}
			if !citationEqualV2(c.Values[id], want) {
				fails = append(fails, c.Item+": the value cited for "+id+" is not the value shown")
				continue
			}
			seen = append(seen, want.Text)
		}
		if !other {
			fails = append(fails, c.Item+": names only the alternative chosen")
		}
		for _, x := range seen {
			if x != seen[0] {
				differ = true
			}
		}
	}
	if !differ {
		fails = append(fails, "no entry shows a difference")
	}
	return len(fails) == 0, fails
}
