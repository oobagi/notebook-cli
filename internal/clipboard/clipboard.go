package clipboard

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Copy copies text to the system clipboard.
// Tries the system clipboard first (more reliable), falls back to OSC 52
// (works over SSH).
func Copy(text string) error {
	if err := copySystem(text); err == nil {
		return nil
	}
	return copyOSC52(text, os.Stdout)
}

// Read returns text from the system clipboard.
func Read() (string, error) {
	return readSystem()
}

// copyOSC52 writes an OSC 52 escape sequence to w.
// Format: \x1b]52;c;<base64>\x07
func copyOSC52(text string, w io.Writer) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	_, err := fmt.Fprintf(w, "\x1b]52;c;%s\x07", encoded)
	return err
}

// copySystem uses platform-specific commands to copy text to the clipboard.
// macOS: pbcopy
// Linux: xclip -selection clipboard
func copySystem(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// readSystem uses platform-specific commands to read text from the clipboard.
// macOS: pbpaste
// Linux: xclip -selection clipboard -o
func readSystem() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
