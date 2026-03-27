# Contributing a Community Glamour Style

A community style is a JSON file that controls how markdown renders in the terminal. Each style defines colors for headings, code blocks, links, and other markdown elements using [Glamour](https://github.com/charmbracelet/glamour), the library Notebook uses for rendering.

## Create Your Style

1. Copy an existing style as a starting point:

   ```bash
   cp styles/community/gruvbox.json styles/community/your-style-name.json
   ```

2. Open your new file and change the colors to match your palette.

3. Save it with a **lowercase kebab-case** name -- no spaces, no underscores. Examples:
   - `solarized-dark.json`
   - `catppuccin-mocha.json`
   - `dracula.json`

## JSON Structure

Here are the key blocks you can customize:

| Block | What it controls |
|-------|-----------------|
| `document` | Base text color and margin for the entire rendered output |
| `heading`, `h1`, `h2`, `h3` | Heading styles -- color, bold, prefix text |
| `code` | Inline code color |
| `code_block` | Fenced code block color, margin, and syntax highlighting via `chroma.theme` |
| `link` | URL color and underline |
| `link_text` | Clickable link text color |
| `strong` | Bold text color |
| `emph` | Italic text color |
| `list` / `item` | List indentation and bullet prefix |

A minimal style must include at least a `document` block. All color values are hex strings (e.g., `"#EBDBB2"`).

For the `code_block.chroma.theme` value, pick a theme from the [Chroma style gallery](https://xyproto.github.io/splash/docs/).

## Test Locally

Run the style validation tests from the repo root:

```bash
go test ./styles/...
```

The tests check that your JSON is valid and that the filename is kebab-case.

## Preview Your Style

1. Run `notebook` to open the TUI browser.
2. Press `t` to open the theme picker.
3. Your style appears under the **community** section.

## Submit a PR

1. Your PR should add **one file**: `styles/community/your-style-name.json`.
2. Tests pass (`go test ./styles/...`).
3. Include a brief description of the color palette or theme that inspired your style.

That's it. Thank you for contributing!

## References

- [Glamour styles reference](https://github.com/charmbracelet/glamour/tree/master/styles)
- [Chroma style gallery](https://xyproto.github.io/splash/docs/)
