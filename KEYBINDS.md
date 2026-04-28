# grotto — Keybindings

## Global

| Key | Action |
|-----|--------|
| Ctrl+Q | Quit |
| Ctrl+B | Toggle sidebar |
| Ctrl+` / F3 | Toggle terminal |
| Ctrl+Shift+A / F4 | Toggle AI panel |
| Ctrl+Shift+L | Send editor selection (or whole file) to AI panel |
| Ctrl+P / F1 | Fuzzy file finder |
| Ctrl+Shift+P / F2 | Command palette |
| F5 | Show keyboard shortcuts |
| Ctrl+1/2/3/4 | Focus pane 1-4 (or sidebar/terminal/AI when single pane) |
| Esc | Focus sidebar (from editor) |

All of these are also available as clickable buttons in the title bar (including **? Keys** to open this reference).

## Editor — Navigation

| Key | Action |
|-----|--------|
| ↑ ↓ ← → | Move cursor |
| Home / End | Start / end of line |
| Ctrl+← / Ctrl+→ | Word left / right |
| PgUp / PgDn | Page up / down |
| Ctrl+G | Go to line |

## Editor — Selection

| Key | Action |
|-----|--------|
| Shift+↑↓←→ | Extend selection |
| Shift+Home/End | Select to start/end of line |
| Ctrl+Shift+←/→ | Select word left/right |
| Shift+PgUp/PgDn | Select page up/down |
| Ctrl+A | Select all |
| Double-click | Select word |
| Triple-click | Select line |
| Shift+click | Extend selection to click |
| Click+drag | Drag selection |

## Editor — Editing

| Key | Action |
|-----|--------|
| Type | Insert text (replaces selection if active) |
| Enter | New line (auto-indent) |
| Backspace | Delete left (or selection) |
| Delete | Delete right (or selection) |
| Tab | Insert 4 spaces (or indent selection) |
| Shift+Tab | Dedent line/selection |
| Ctrl+D | Duplicate line |
| Ctrl+Z | Undo |
| Ctrl+Y | Redo |
| Ctrl+S | Save |

## Editor — Clipboard

| Key | Action |
|-----|--------|
| Ctrl+C | Copy selection |
| Ctrl+X | Cut selection |
| Ctrl+V | Paste |

## Editor — Search

| Key | Action |
|-----|--------|
| Ctrl+F | Find |
| Ctrl+H | Find & replace |
| Enter / ↓ / Ctrl+N | Next match |
| ↑ / Ctrl+P | Previous match |
| Tab | Switch find ↔ replace field |
| Enter (in replace) | Replace one |
| Ctrl+Shift+Enter | Replace all |
| Esc | Close search |

## Tabs

| Key | Action |
|-----|--------|
| Ctrl+Tab | Next tab |
| Ctrl+Shift+Tab | Previous tab |
| Ctrl+W | Close tab |
| Click tab | Switch to tab |
| Middle-click tab | Close tab |

## Split Panes

| Key | Action |
|-----|--------|
| Ctrl+\ | Split right |
| Ctrl+Shift+\ | Split down |
| Ctrl+Shift+W | Close pane |
| Ctrl+1/2/3/4 | Focus pane by number |
| Click in pane | Focus pane |

## Mouse

| Action | Effect |
|--------|--------|
| Click | Place cursor |
| Click+drag | Select text |
| Scroll wheel | Scroll (3 lines) |
| Click sidebar item | Open file / expand folder |
| Scroll in sidebar | Scroll file tree |
| Click in terminal | Focus terminal panel |

## Terminal

When the terminal panel is focused, most keys are forwarded to the shell.
Global keybinds (Ctrl+Q, Ctrl+`, Ctrl+B, Ctrl+Shift+A) still work.

| Key | Action |
|-----|--------|
| Ctrl+` | Toggle terminal panel (focuses it when opened) |
| All keys | Forwarded to shell when terminal is focused |
| Ctrl+C/D/Z | Sent to shell (signal/EOF/suspend) |
