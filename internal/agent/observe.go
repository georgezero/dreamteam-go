package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// Observation projections (V0.3b). A sectioned or compact observation is
// cut from the JSON of the full observation the session returns: the same
// bytes, some keys left out, the order kept. It reads nothing else, so it
// can show nothing the full observation does not, and every value it
// shows is the full observation's.

// headerKeys are in every projection: what the observation is and when.
var headerKeys = []string{"schema", "time", "day", "weekday", "date", "clock"}

// compactOmits are the record listings a compact observation leaves out,
// as dotted paths: history, not the current state. Every count, rate,
// aggregate, active alert and unread worklist item stays.
var compactOmits = []struct {
	Path, What string
}{
	{"finance.claims.openClaims", "the open claims, one by one (the counts and aging stay)"},
	{"finance.journal", "the latest journal entries (the totals and balances stay)"},
	{"worklist.recent", "the latest finalized reports (the unread and reading items stay)"},
	{"alerts.resolved", "the resolved alerts (the active ones stay)"},
}

// Section describes one section of the observation.
type Section struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`             // object, array, string, integer
	Fields      []string `json:"fields,omitempty"` // an object's keys (or an array element's), in order
	Header      bool     `json:"header,omitempty"` // always included
	CompactOmit []string `json:"compactOmits,omitempty"`
}

// Sections lists the observation's sections from api.Observation itself.
func Sections() []Section {
	var out []Section
	t := reflect.TypeOf(api.Observation{})
	for i := range t.NumField() {
		f := t.Field(i)
		name := jsonName(f)
		if name == "" {
			continue
		}
		s := Section{Name: name, Header: slices.Contains(headerKeys, name)}
		ft := f.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.Struct:
			s.Type, s.Fields = "object", structFields(ft)
		case reflect.Slice:
			s.Type = "array"
			if e := ft.Elem(); e.Kind() == reflect.Struct {
				s.Fields = structFields(e)
			}
		case reflect.String:
			s.Type = "string"
		default:
			s.Type = "integer"
		}
		for _, o := range compactOmits {
			if strings.HasPrefix(o.Path, name+".") {
				s.CompactOmit = append(s.CompactOmit, o.Path)
			}
		}
		out = append(out, s)
	}
	return out
}

func jsonName(f reflect.StructField) string {
	tag := strings.Split(f.Tag.Get("json"), ",")[0]
	if tag == "-" || !f.IsExported() {
		return ""
	}
	if tag == "" {
		return f.Name
	}
	return tag
}

func structFields(t reflect.Type) []string {
	var out []string
	for i := range t.NumField() {
		if n := jsonName(t.Field(i)); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// jsonObject is a JSON object with its keys in order.
type jsonObject struct {
	keys []string
	vals map[string]json.RawMessage
}

func decodeObject(raw []byte) (*jsonObject, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, fmt.Errorf("not an object")
	}
	o := &jsonObject{vals: map[string]json.RawMessage{}}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		k := t.(string)
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		o.keys = append(o.keys, k)
		o.vals[k] = v
	}
	return o, nil
}

func (o *jsonObject) encode() json.RawMessage {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		b.Write(kb)
		b.WriteByte(':')
		b.Write(o.vals[k])
	}
	b.WriteByte('}')
	return b.Bytes()
}

// remove deletes the dotted path from o, returning the removed value
// (nil if absent).
func (o *jsonObject) remove(path []string) (json.RawMessage, error) {
	v, ok := o.vals[path[0]]
	if !ok {
		return nil, nil
	}
	if len(path) == 1 {
		delete(o.vals, path[0])
		o.keys = slices.DeleteFunc(o.keys, func(k string) bool { return k == path[0] })
		return v, nil
	}
	if bytes.Equal(v, []byte("null")) {
		return nil, nil
	}
	sub, err := decodeObject(v)
	if err != nil {
		return nil, err
	}
	got, err := sub.remove(path[1:])
	if err == nil && got != nil {
		o.vals[path[0]] = sub.encode()
	}
	return got, err
}

// Projected is a sectioned or compact observation: the observation's
// selected sections (and the header), and, when compact, how many entries
// of each left-out listing there were.
type Projected struct {
	Observation json.RawMessage `json:"observation"`
	Sections    []string        `json:"sections"`
	Compact     bool            `json:"compact"`
	Omitted     map[string]int  `json:"omitted,omitempty"`
}

// project cuts obs to the sections named (all when none) and, when
// compact, leaves out compactOmits.
func project(obs api.Observation, sections []string, compact bool) (Projected, *Error) {
	all := Sections()
	names := make([]string, len(all))
	for i, s := range all {
		names[i] = s.Name
	}
	for i, s := range sections {
		if !slices.Contains(names, s) {
			return Projected{}, &Error{Code: ErrUnknownSection, Message: fmt.Sprintf("no section %q; the sections are %v (op sections)", s, names), Field: "sections", Index: &i}
		}
		if slices.Contains(sections[:i], s) {
			return Projected{}, &Error{Code: ErrInvalidArgs, Message: fmt.Sprintf("section %q twice", s), Field: "sections", Index: &i}
		}
	}
	raw, err := json.Marshal(obs)
	if err != nil {
		return Projected{}, &Error{Code: "internal_error", Message: err.Error()}
	}
	full, err := decodeObject(raw)
	if err != nil {
		return Projected{}, &Error{Code: "internal_error", Message: err.Error()}
	}
	keep := func(k string) bool {
		return len(sections) == 0 || slices.Contains(sections, k) || slices.Contains(headerKeys, k)
	}
	out := &jsonObject{vals: map[string]json.RawMessage{}}
	p := Projected{Sections: []string{}, Compact: compact}
	for _, k := range full.keys {
		if keep(k) {
			out.keys = append(out.keys, k)
			out.vals[k] = full.vals[k]
			p.Sections = append(p.Sections, k)
		}
	}
	if compact {
		p.Omitted = map[string]int{}
		for _, o := range compactOmits {
			v, err := out.remove(strings.Split(o.Path, "."))
			if err != nil {
				return Projected{}, &Error{Code: "internal_error", Message: err.Error()}
			}
			if v != nil {
				var xs []json.RawMessage
				json.Unmarshal(v, &xs)
				p.Omitted[o.Path] = len(xs)
			}
		}
	}
	p.Observation = out.encode()
	return p, nil
}
