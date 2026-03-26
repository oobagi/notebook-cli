package cmd

import (
	"fmt"
	"io"
)

func printSuccess(w io.Writer, msg string) {
	fmt.Fprintf(w, "  \u2713 %s\n", msg)
}

func printError(w io.Writer, msg string) {
	fmt.Fprintf(w, "  \u2717 %s\n", msg)
}

func printInfo(w io.Writer, msg string) {
	fmt.Fprintf(w, "  \u2192 %s\n", msg)
}

func printWarning(w io.Writer, msg string) {
	fmt.Fprintf(w, "  ! %s\n", msg)
}
