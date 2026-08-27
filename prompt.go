package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

// exitOnEOF stops the program cleanly when stdin is closed mid-prompt
// (e.g. piped input ran out, or the terminal disconnected), instead of
// letting callers loop forever on a ReadString that keeps returning "".
func exitOnEOF(err error) {
	if err != nil {
		fmt.Println("\n(no more input -- exiting)")
		os.Exit(0)
	}
}

func promptLine(label string) string {
	fmt.Printf("%s: ", label)
	line, err := stdinReader.ReadString('\n')
	exitOnEOF(err)
	return strings.TrimSpace(line)
}

// promptDefault shows def as the pre-filled value; pressing Enter keeps it.
func promptDefault(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := stdinReader.ReadString('\n')
	exitOnEOF(err)
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func confirm(label string, def bool) bool {
	suffix := "[y/N]"
	if def {
		suffix = "[Y/n]"
	}
	fmt.Printf("%s %s: ", label, suffix)
	line, err := stdinReader.ReadString('\n')
	exitOnEOF(err)
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}
