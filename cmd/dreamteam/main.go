// Command dreamteam runs the DreamTeam outpatient radiology practice
// simulator.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

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
  dreamteam [play] [-seed N]   play in the terminal (default)
  dreamteam version            print the version
`)
}

func play(args []string) int {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	seed := fs.Int64("seed", 1, "simulation seed")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	m := tui.New(session.NewDemo(*seed))
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dreamteam:", err)
		return 1
	}
	return 0
}
