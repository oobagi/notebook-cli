package ui

// HandlePickerKey processes a key event for a Picker.
// Returns handled=true if the picker consumed the key, and closed=true
// if the picker closed itself (backspace on empty filter, or esc).
// When the key is "enter", the caller should check p.Selected() for the
// chosen item — the picker does NOT close itself on enter.
func HandlePickerKey(p *Picker, key string, text string, code rune) (handled, closed bool) {
	if !p.Visible {
		return false, false
	}

	switch key {
	case "up":
		p.MoveUp()
		return true, false
	case "down":
		p.MoveDown()
		return true, false
	case "enter":
		// Caller handles selection; picker stays open until caller closes it.
		return true, false
	case "esc":
		p.Close()
		return true, true
	case "backspace":
		if !p.DeleteFilterRune() {
			p.Close()
			return true, true
		}
		return true, false
	case "alt+backspace", "ctrl+w":
		if !p.DeleteFilterWord() {
			p.Close()
			return true, true
		}
		return true, false
	case "ctrl+u":
		if !p.ClearFilter() {
			p.Close()
			return true, true
		}
		return true, false
	default:
		if len(text) > 0 {
			for _, r := range text {
				p.AddFilterRune(r)
			}
			return true, false
		}
	}

	// Swallow unknown keys while picker is open.
	return true, false
}
