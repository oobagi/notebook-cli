#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
RESET='\033[0m'

echo ""
echo -e "${BOLD}notebook uninstaller${RESET}"
echo "════════════════════════════════════"
echo ""

# --- Detect installations ---

found=0

# Homebrew
if brew list notebook &>/dev/null; then
    echo -e "  ${GREEN}✓${RESET} Homebrew: $(brew --prefix)/Cellar/notebook"
    found=1
else
    echo -e "  - Homebrew: not installed"
fi

# go install
gobin="${GOPATH:-$HOME/go}/bin/notebook"
if [[ -f "$gobin" ]]; then
    echo -e "  ${GREEN}✓${RESET} go install: $gobin"
    found=1
else
    echo -e "  - go install: not installed"
fi

# Manual binary
manual=""
if command -v notebook &>/dev/null; then
    loc=$(command -v notebook)
    if [[ "$loc" != "$(brew --prefix 2>/dev/null)/bin/notebook" && "$loc" != "$gobin" ]]; then
        echo -e "  ${GREEN}✓${RESET} Manual binary: $loc"
        manual="$loc"
        found=1
    fi
fi

echo ""

# Config
config_dir="$HOME/.config/notebook"
if [[ -d "$config_dir" ]]; then
    echo -e "  ${GREEN}✓${RESET} Config: $config_dir"
else
    echo -e "  - Config: none"
fi

# Data
data_dir="$HOME/.notebook"
if [[ -d "$data_dir" ]]; then
    count=$(find "$data_dir" -name '*.md' 2>/dev/null | wc -l | tr -d ' ')
    echo -e "  ${YELLOW}!${RESET} Notes: $data_dir ($count markdown files)"
else
    echo -e "  - Notes: none"
fi

echo ""

if [[ $found -eq 0 ]]; then
    echo "No notebook installation found."
    exit 0
fi

# --- Confirm ---

echo -e "${BOLD}This will remove:${RESET}"
brew list notebook &>/dev/null && echo "  • Homebrew formula + binary"
[[ -f "$gobin" ]] && echo "  • go install binary at $gobin"
[[ -n "$manual" ]] && echo "  • Manual binary at $manual"
echo "  • Config directory ($config_dir)"
echo ""
echo -e "${YELLOW}Your notes in $data_dir will NOT be deleted.${RESET}"
echo -e "To remove notes too, pass ${BOLD}--all${RESET}"
echo ""

read -rp "Proceed? [y/N] " confirm
if [[ "$confirm" != [yY] ]]; then
    echo "Aborted."
    exit 0
fi

echo ""

# --- Remove ---

# Homebrew
if brew list notebook &>/dev/null; then
    echo "Removing Homebrew formula..."
    brew uninstall notebook
    # Remove tap if no other formulae from it
    if brew tap-info oobagi/tap &>/dev/null; then
        tap_formulae=$(brew list --formula | grep -c "oobagi/tap" 2>/dev/null || true)
        if [[ "$tap_formulae" -eq 0 ]]; then
            echo "Removing tap oobagi/tap..."
            brew untap oobagi/tap 2>/dev/null || true
        fi
    fi
    echo -e "  ${GREEN}✓${RESET} Homebrew removed"
fi

# go install
if [[ -f "$gobin" ]]; then
    rm "$gobin"
    echo -e "  ${GREEN}✓${RESET} go install binary removed"
fi

# Manual binary
if [[ -n "$manual" ]]; then
    rm "$manual"
    echo -e "  ${GREEN}✓${RESET} Manual binary removed"
fi

# Config
if [[ -d "$config_dir" ]]; then
    rm -rf "$config_dir"
    echo -e "  ${GREEN}✓${RESET} Config removed"
fi

# Notes (only with --all)
if [[ "${1:-}" == "--all" ]] && [[ -d "$data_dir" ]]; then
    echo -e "${RED}Removing all notes in $data_dir...${RESET}"
    rm -rf "$data_dir"
    echo -e "  ${GREEN}✓${RESET} Notes removed"
fi

echo ""
echo -e "${GREEN}${BOLD}Uninstall complete.${RESET}"
echo ""
echo "To reinstall via Homebrew:"
echo "  brew tap oobagi/tap"
echo "  brew install notebook"
echo ""
