package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// t90Cmd evaluates a saved session's plan (V0.2_DESIGN §13) under the
// evaluation policy in force: the T90 metric vector, the service gates and
// the verdict, against no action on the same seed. The session is replayed
// first, every digest verified.
func t90Cmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("t90", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "text (default) or json (the evaluation with the policy's full report)")
	pos, err := parseFlags(fs, args)
	if err != nil || len(pos) != 1 || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "usage: dreamteam t90 FILE [-format text|json]")
		return exitInvalid
	}
	s, rep, err := session.Load(pos[0], session.ReplayOptions{})
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		if rep != nil && len(rep.Mismatches) > 0 {
			fmt.Fprint(stderr, rep)
		}
		if errors.Is(err, session.ErrIncompatible) {
			return exitIncompatible
		}
		return exitReplay
	}
	t, err := s.T90()
	switch {
	case errors.Is(err, session.ErrT90Scenario):
		fmt.Fprintln(stderr, "dreamteam:", err, "(dreamteam -scenario sustainable)")
		return exitIncompatible
	case errors.Is(err, session.ErrT90Rules):
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitIncompatible
	case errors.Is(err, eval.ErrSealed), errors.Is(err, session.ErrT90Started):
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitInvalid
	case err != nil:
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	if *format == "json" {
		raw, err := json.MarshalIndent(t, "", " ")
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		fmt.Fprintf(stdout, "%s\n", raw)
		return exitOK
	}
	for _, l := range t.Lines(160) {
		fmt.Fprintln(stdout, l)
	}
	return exitOK
}
