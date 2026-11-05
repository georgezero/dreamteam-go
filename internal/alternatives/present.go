package alternatives

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// The presentation template, dreamteam.alternatives-presentation/v1: fixed
// text, pinned by SHA-256, rendering only the agent audience. It is never
// sent to a model in V0.13b.
const (
	PresentationID     = "dreamteam.alternatives-presentation/v1"
	PresentationSha256 = "5bf586cf12c83c157a8f31db05cd8f5cc68c0503b3bf99b1a89102198313b89f"
)

//go:embed text/presentation-v1.tmpl
var presentationText []byte

var presentation = template.Must(template.New("p").Funcs(template.FuncMap{
	"inc":  func(i int) int { return i + 1 },
	"json": func(v any) (string, error) { b, err := json.Marshal(v); return string(b), err },
	"val":  formatValue,
}).Parse(string(presentationText)))

// formatValue is a value as shown: cents as dollars, counts and days as
// integers, dates and text as they are.
func formatValue(v Value) string {
	switch v.Unit {
	case UnitCents:
		return money(*v.Number)
	case UnitCount:
		return fmt.Sprint(*v.Number)
	case UnitDays:
		return fmt.Sprintf("%d days", *v.Number)
	case UnitSec:
		return fmt.Sprintf("%d s", *v.Number)
	case UnitDate:
		return v.Date
	}
	if v.Number != nil {
		return v.Text + " (" + fmt.Sprint(*v.Number) + ")"
	}
	return v.Text
}

func money(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	d := fmt.Sprint(c / 100)
	var b strings.Builder
	for i, r := range d {
		if i > 0 && (len(d)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return fmt.Sprintf("%s$%s.%02d", sign, b.String(), c%100)
}

// Summary is the first level the agent sees: each alternative's ID,
// actions, validation and the size of its detail.
func Summary(a *Agent) (string, error) {
	if err := a.check(); err != nil {
		return "", err
	}
	var b bytes.Buffer
	err := presentation.ExecuteTemplate(&b, "summary", a)
	return b.String(), err
}

// Detail is one alternative's detail, delivered on inspect:<id>.
func Detail(a *Agent, id string) (string, error) {
	if err := a.check(); err != nil {
		return "", err
	}
	for _, x := range a.Alternatives {
		if x.ID == id {
			var b bytes.Buffer
			err := presentation.ExecuteTemplate(&b, "detail", x)
			return b.String(), err
		}
	}
	return "", fmt.Errorf("%w: no alternative %q", ErrContract, id)
}
