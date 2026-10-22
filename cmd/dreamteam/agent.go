package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// agentCmd serves the agent protocol (docs/AGENT_PROTOCOL.md) on standard
// input and output (-stdio), or runs a scripted policy through it
// (-policy). Only protocol responses (-stdio) or the run's summary
// (-policy) go to stdout; diagnostics go to stderr.
func agentCmd(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(stderr)
	stdio := fs.Bool("stdio", false, "serve the protocol: one JSON request per line on stdin, one response per line on stdout")
	policy := fs.String("policy", "", "run a scripted policy through the protocol: noop or greedy")
	days := fs.Int("days", 30, "(-policy) days to play")
	seed := fs.Int64("seed", 1, "simulation seed")
	scen := fs.String("scenario", scenario.Default, "bundled scenario ID or .json file")
	load := fs.String("load", "", "start from a saved session file (replayed and verified)")
	transcript := fs.String("transcript", "", "(-policy) write every request and response line to this file")
	save := fs.String("save", "", "(-policy) save the session to this file at the end")
	pos, err := parseFlags(fs, args)
	if err != nil || len(pos) > 0 || *stdio == (*policy != "") || *days < 1 || *days > 3650 {
		fmt.Fprintln(stderr, "usage: dreamteam agent --stdio [-scenario ID|FILE] [-seed N] [-load FILE]\n"+
			"       dreamteam agent -policy noop|greedy [-days N] [-scenario ID|FILE] [-seed N] [-load FILE] [-transcript FILE] [-save FILE]")
		return exitInvalid
	}
	var s *session.Session
	ref := *scen
	if *load != "" {
		var rep *session.Report
		s, rep, err = session.Load(*load, session.ReplayOptions{})
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			if rep != nil {
				fmt.Fprint(stderr, rep)
			}
			return exitReplay
		}
		ref = *load
	} else if s, err = session.Open(*scen, *seed); err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	h := agent.New(s)
	h.Ref = ref
	if *stdio {
		fmt.Fprintf(stderr, "dreamteam agent: %s · scenario %s · seed %d · ready\n", agent.Protocol, ref, s.Seed())
		if err := agent.ServeLines(stdin, stdout, h); err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		return exitOK
	}
	var p agent.Policy
	switch *policy {
	case "noop":
		p = agent.Noop{}
	case "greedy":
		p = agent.Greedy{}
	default:
		fmt.Fprintf(stderr, "dreamteam: unknown policy %q (noop or greedy)\n", *policy)
		return exitInvalid
	}
	c := &agent.Client{H: h}
	if *transcript != "" {
		f, err := os.Create(*transcript)
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		defer f.Close()
		c.Transcript = f
	}
	sum, err := agent.Run(c, p, *days)
	if err == nil && *save != "" {
		err = c.Call("save", map[string]any{"path": *save}, nil)
	}
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	raw, _ := json.Marshal(sum)
	fmt.Fprintf(stdout, "%s\n", raw)
	return exitOK
}
