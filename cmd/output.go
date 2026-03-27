package cmd

import (
	"fmt"
	"io"
	"os"
)

func colorEnabled() bool {
	_, noColor := os.LookupEnv("NO_COLOR")
	return !noColor
}

func printSuccess(w io.Writer, msg string) {
	if colorEnabled() {
		fmt.Fprintf(w, "  \x1b[32m\u2713\x1b[0m %s\n", msg)
	} else {
		fmt.Fprintf(w, "  \u2713 %s\n", msg)
	}
}

func printError(w io.Writer, msg string) {
	if colorEnabled() {
		fmt.Fprintf(w, "  \x1b[31m\u2717\x1b[0m %s\n", msg)
	} else {
		fmt.Fprintf(w, "  \u2717 %s\n", msg)
	}
}

func printInfo(w io.Writer, msg string) {
	if colorEnabled() {
		fmt.Fprintf(w, "  \x1b[2m\u2192\x1b[0m %s\n", msg)
	} else {
		fmt.Fprintf(w, "  \u2192 %s\n", msg)
	}
}

func printWarning(w io.Writer, msg string) {
	if colorEnabled() {
		fmt.Fprintf(w, "  \x1b[33m!\x1b[0m %s\n", msg)
	} else {
		fmt.Fprintf(w, "  ! %s\n", msg)
	}
}

// PrintErrorStderr prints a styled error message to stderr.
// Exported so main.go can use it for top-level error handling.
func PrintErrorStderr(msg string) {
	printError(os.Stderr, msg)
}
