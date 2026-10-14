// Command dreamteam runs the DreamTeam outpatient radiology practice
// simulator.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/tui"
)

const version = "0.1.0-dev"

func main() {
	args := os.Args[1:]
	cmd := "play"
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "play":
		os.Exit(play(args))
	case "scenario":
		os.Exit(scenarioCmd(args))
	case "version":
		fmt.Println("dreamteam", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "dreamteam: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:
  dreamteam [play] [-scenario ID|FILE] [-seed N]   play in the terminal (default)
  dreamteam scenario list                          list bundled scenarios
  dreamteam scenario validate ID|FILE              check a scenario
  dreamteam scenario coverage ID|FILE              staffed vs open hours, fortnight rota
  dreamteam version                                print the version
`)
}

func play(args []string) int {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	seed := fs.Int64("seed", 1, "simulation seed")
	scen := fs.String("scenario", scenario.Default, "bundled scenario ID or .json file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s, err := session.Open(*scen, *seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dreamteam:", err)
		return 1
	}
	m := tui.New(s)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dreamteam:", err)
		return 1
	}
	return 0
}

func scenarioCmd(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "list":
		for _, id := range scenario.List() {
			fmt.Println(id)
		}
		return 0
	case "validate", "coverage":
		if len(args) != 2 {
			usage()
			return 2
		}
		cfg, err := scenario.Resolve(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if args[0] == "validate" {
			fmt.Printf("%s: ok (%d rooms, %d employees)\n", cfg.ID, len(cfg.Rooms), len(cfg.Employees))
		} else {
			fmt.Print(scenario.Report(cfg))
		}
		return 0
	}
	usage()
	return 2
}
