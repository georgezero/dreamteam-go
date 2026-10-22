package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// dreamteam t90 replays a saved session and prints the same evaluation the
// session computes (JSON), or its text; other scenarios are refused as
// incompatible, and bad usage as invalid input.
func TestT90Command(t *testing.T) {
	if testing.Short() {
		t.Skip("runs six 200-day worlds")
	}
	dir := t.TempDir()
	s, err := session.Open("sustainable", 906)
	if err != nil {
		t.Fatal(err)
	}
	s.Apply(api.Action{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E07", Effective: "2026-10-27", SiteAllocationBP: 6250})
	s.AdvanceTo(s.Now().Add(40 * simtime.Day))
	file := filepath.Join(dir, "s.session.json")
	if err := s.Save(file); err != nil {
		t.Fatal(err)
	}
	want, err := s.T90()
	if err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if code := t90Cmd([]string{file, "-format", "json"}, &out, &errs); code != exitOK {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	w, _ := json.MarshalIndent(want, "", " ")
	if strings.TrimSpace(out.String()) != string(w) {
		t.Fatal("the command's evaluation differs from the session's")
	}
	out.Reset()
	if code := t90Cmd([]string{file}, &out, &errs); code != exitOK || out.String() != strings.Join(want.Lines(160), "\n")+"\n" {
		t.Fatalf("text: exit %d", code)
	}
	if !strings.Contains(out.String(), "PROJECTED: you are on day 40.0") {
		t.Error("the projection is not stated")
	}
	e, err := session.Open("established", 1)
	if err != nil {
		t.Fatal(err)
	}
	e.AdvanceTo(e.Now().Add(simtime.Day))
	other := filepath.Join(dir, "e.session.json")
	if err := e.Save(other); err != nil {
		t.Fatal(err)
	}
	errs.Reset()
	if code := t90Cmd([]string{other}, &out, &errs); code != exitIncompatible || !strings.Contains(errs.String(), "sustainable") {
		t.Errorf("established: exit %d, %s", code, errs.String())
	}
	if code := t90Cmd([]string{file, "-format", "xml"}, &out, &errs); code != exitInvalid {
		t.Errorf("bad format: exit %d", code)
	}
}
