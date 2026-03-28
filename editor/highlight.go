package editor

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter caches syntax tokens per line.
type Highlighter struct {
	lexer     chroma.Lexer
	style     *chroma.Style
	cache     map[int][]StyledSpan        // line number → spans
	ansiCache map[chroma.TokenType]string // token type → ANSI prefix
}

// StyledSpan is a chunk of text with a pre-computed ANSI prefix for fast rendering.
type StyledSpan struct {
	Text  string
	Style lipgloss.Style
	ANSI  string // pre-computed "\x1b[...m" prefix (empty = no styling)
}

const ansiReset = "\x1b[0m"

func NewHighlighter(filePath string) *Highlighter {
	h := &Highlighter{
		style:     styles.Get("dracula"),
		cache:     make(map[int][]StyledSpan),
		ansiCache: make(map[chroma.TokenType]string),
	}
	if filePath != "" {
		h.lexer = lexers.Match(filepath.Base(filePath))
	}
	if h.lexer == nil {
		h.lexer = lexers.Fallback
	}
	h.lexer = chroma.Coalesce(h.lexer)
	return h
}

// InvalidateLine clears the cache for a line (call on edit).
func (h *Highlighter) InvalidateLine(line int) {
	delete(h.cache, line)
}

// InvalidateAll clears the entire cache.
func (h *Highlighter) InvalidateAll() {
	h.cache = make(map[int][]StyledSpan)
}

// Highlight returns styled spans for a single line.
func (h *Highlighter) Highlight(lineNum int, text string) []StyledSpan {
	if spans, ok := h.cache[lineNum]; ok {
		return spans
	}

	iter, err := h.lexer.Tokenise(nil, text+"\n")
	if err != nil {
		span := StyledSpan{Text: text, Style: lipgloss.NewStyle()}
		h.cache[lineNum] = []StyledSpan{span}
		return h.cache[lineNum]
	}

	var spans []StyledSpan
	for _, tok := range iter.Tokens() {
		// Strip trailing newlines added by our tokenisation trick
		t := strings.TrimSuffix(tok.Value, "\n")
		if t == "" {
			continue
		}
		s := h.tokenStyle(tok.Type)
		a := h.tokenANSI(tok.Type)
		spans = append(spans, StyledSpan{Text: t, Style: s, ANSI: a})
	}
	if lineNum >= 0 {
		h.cache[lineNum] = spans
	}
	return spans
}

// RenderLine returns the fully styled string for a line.
func (h *Highlighter) RenderLine(lineNum int, text string) string {
	spans := h.Highlight(lineNum, text)
	var b strings.Builder
	for _, sp := range spans {
		if sp.ANSI != "" {
			b.WriteString(sp.ANSI)
			b.WriteString(sp.Text)
			b.WriteString(ansiReset)
		} else {
			b.WriteString(sp.Text)
		}
	}
	return b.String()
}

func (h *Highlighter) tokenStyle(tt chroma.TokenType) lipgloss.Style {
	s := lipgloss.NewStyle()
	entry := h.style.Get(tt)
	if entry.Colour.IsSet() {
		s = s.Foreground(lipgloss.Color(entry.Colour.String()))
	}
	if entry.Bold == chroma.Yes {
		s = s.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		s = s.Italic(true)
	}
	return s
}

// tokenANSI returns a cached raw ANSI prefix for a token type.
func (h *Highlighter) tokenANSI(tt chroma.TokenType) string {
	if a, ok := h.ansiCache[tt]; ok {
		return a
	}
	entry := h.style.Get(tt)
	a := buildANSI(entry)
	h.ansiCache[tt] = a
	return a
}

// buildANSI creates a raw ANSI escape prefix from a chroma style entry.
func buildANSI(entry chroma.StyleEntry) string {
	var parts []string
	if entry.Bold == chroma.Yes {
		parts = append(parts, "1")
	}
	if entry.Italic == chroma.Yes {
		parts = append(parts, "3")
	}
	if entry.Colour.IsSet() {
		r, g, b := entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue()
		parts = append(parts, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
	}
	if len(parts) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

// FastRender writes styled text using raw ANSI codes (no lipgloss overhead).
func FastRender(text, ansiPrefix string) string {
	if ansiPrefix == "" {
		return text
	}
	return ansiPrefix + text + ansiReset
}

// BuildANSIPrefix builds a raw ANSI prefix from fg/bg hex colors and attributes.
func BuildANSIPrefix(fg, bg string, bold, italic, reverse bool) string {
	var parts []string
	if bold {
		parts = append(parts, "1")
	}
	if italic {
		parts = append(parts, "3")
	}
	if reverse {
		parts = append(parts, "7")
	}
	if fg != "" {
		var r, g, b uint8
		fmt.Sscanf(fg, "#%02x%02x%02x", &r, &g, &b)
		parts = append(parts, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
	}
	if bg != "" {
		var r, g, b uint8
		fmt.Sscanf(bg, "#%02x%02x%02x", &r, &g, &b)
		parts = append(parts, fmt.Sprintf("48;2;%d;%d;%d", r, g, b))
	}
	if len(parts) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}
