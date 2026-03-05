package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AgusRdz/chop/filters"
	"github.com/AgusRdz/chop/tracking"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: chop <command> [args...]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--version", "version":
		fmt.Printf("chop %s\n", version)
		return
	case "gain":
		runGain(os.Args[2:])
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	filter := filters.Get(command, args)

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "chop: failed to run %s: %v\n", command, err)
			os.Exit(1)
		}
	}

	raw := string(output)
	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}

	if filter != nil {
		filtered, ferr := filter(raw)
		if ferr != nil {
			fmt.Print(raw) // graceful fallback
			trackSilent(fullCmd, raw, raw)
		} else {
			fmt.Print(filtered)
			trackSilent(fullCmd, raw, filtered)
		}
	} else {
		// Auto-detect compression for unrecognized commands
		autoFiltered, aerr := filters.AutoDetect(raw)
		if aerr != nil || autoFiltered == raw {
			fmt.Print(raw)
			trackSilent(fullCmd, raw, raw)
		} else {
			fmt.Print(autoFiltered)
			trackSilent(fullCmd, raw, autoFiltered)
		}
	}

	os.Exit(exitCode)
}

func trackSilent(command, raw, filtered string) {
	rawTokens := tracking.CountTokens(raw)
	filteredTokens := tracking.CountTokens(filtered)
	_ = tracking.Track(command, rawTokens, filteredTokens)
}

func runGain(args []string) {
	showHistory := false
	for _, a := range args {
		if a == "--history" {
			showHistory = true
		}
	}

	if showHistory {
		records, err := tracking.GetHistory(20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read history: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(tracking.FormatHistory(records))
		return
	}

	stats, err := tracking.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to read stats: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(tracking.FormatGain(stats))
}
