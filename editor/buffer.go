package editor

import (
	"os"
	"strings"
	"unicode"
)

type Position struct {
	Line int
	Col  int
}

type Selection struct {
	Anchor Position
	Head   Position
	Active bool
}

type EditType int

const (
	EditInsert EditType = iota
	EditDelete
	EditReplace
)

type Edit struct {
	Type    EditType
	Pos     Position
	Text    string
	OldText string
}

type Buffer struct {
	Lines     []string
	Cursor    Position
	Selection Selection
	FilePath  string
	Dirty     bool
	undoStack []Edit
	redoStack []Edit
}

func NewBuffer(filePath string) (*Buffer, error) {
	b := &Buffer{FilePath: filePath}
	if filePath == "" {
		b.Lines = []string{""}
		return b, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			b.Lines = []string{""}
			return b, nil
		}
		return nil, err
	}
	content := strings.TrimSuffix(string(data), "\n")
	if content == "" {
		b.Lines = []string{""}
	} else {
		b.Lines = strings.Split(content, "\n")
	}
	return b, nil
}

func NewBufferFromString(content string) *Buffer {
	b := &Buffer{}
	if content == "" {
		b.Lines = []string{""}
	} else {
		b.Lines = strings.Split(content, "\n")
	}
	return b
}

// Insert inserts text at pos, handling embedded newlines.
func (b *Buffer) Insert(pos Position, text string) {
	if len(b.Lines) == 0 {
		b.Lines = []string{""}
	}
	b.pushUndo(Edit{Type: EditInsert, Pos: pos, Text: text})

	line := b.Lines[pos.Line]
	before, after := line[:pos.Col], line[pos.Col:]

	parts := strings.Split(text, "\n")
	if len(parts) == 1 {
		b.Lines[pos.Line] = before + text + after
	} else {
		newLines := make([]string, 0, len(b.Lines)+len(parts)-1)
		newLines = append(newLines, b.Lines[:pos.Line]...)
		newLines = append(newLines, before+parts[0])
		for i := 1; i < len(parts)-1; i++ {
			newLines = append(newLines, parts[i])
		}
		newLines = append(newLines, parts[len(parts)-1]+after)
		newLines = append(newLines, b.Lines[pos.Line+1:]...)
		b.Lines = newLines
	}
	b.Dirty = true
}

// Delete removes text from start to end (start inclusive, end exclusive by character).
func (b *Buffer) Delete(start, end Position) {
	if start.Line == end.Line && start.Col == end.Col {
		return
	}
	// Normalize order
	if end.Line < start.Line || (end.Line == start.Line && end.Col < start.Col) {
		start, end = end, start
	}

	deleted := b.textBetween(start, end)
	b.pushUndo(Edit{Type: EditDelete, Pos: start, Text: deleted})

	before := b.Lines[start.Line][:start.Col]
	after := b.Lines[end.Line][end.Col:]
	b.Lines[start.Line] = before + after

	if end.Line > start.Line {
		b.Lines = append(b.Lines[:start.Line+1], b.Lines[end.Line+1:]...)
	}
	b.Dirty = true
}

func (b *Buffer) InsertChar(ch rune) {
	b.Insert(b.Cursor, string(ch))
	b.Cursor.Col += len(string(ch))
	b.clampCursor()
}

func (b *Buffer) Backspace() {
	if b.Cursor.Col == 0 && b.Cursor.Line == 0 {
		return
	}
	var delStart Position
	if b.Cursor.Col > 0 {
		delStart = Position{Line: b.Cursor.Line, Col: b.Cursor.Col - 1}
	} else {
		delStart = Position{Line: b.Cursor.Line - 1, Col: len(b.Lines[b.Cursor.Line-1])}
	}
	b.Delete(delStart, b.Cursor)
	b.Cursor = delStart
	b.clampCursor()
}

func (b *Buffer) DeleteChar() {
	if b.Cursor.Line >= len(b.Lines) {
		return
	}
	var delEnd Position
	if b.Cursor.Col < len(b.Lines[b.Cursor.Line]) {
		delEnd = Position{Line: b.Cursor.Line, Col: b.Cursor.Col + 1}
	} else if b.Cursor.Line < len(b.Lines)-1 {
		delEnd = Position{Line: b.Cursor.Line + 1, Col: 0}
	} else {
		return
	}
	b.Delete(b.Cursor, delEnd)
}

func (b *Buffer) NewLine() {
	line := b.Lines[b.Cursor.Line]
	indent := leadingWhitespace(line)

	b.Insert(b.Cursor, "\n"+indent)
	b.Cursor.Line++
	b.Cursor.Col = len(indent)
	b.clampCursor()
}

func (b *Buffer) LineCount() int {
	return len(b.Lines)
}

func (b *Buffer) Line(n int) string {
	if n < 0 || n >= len(b.Lines) {
		return ""
	}
	return b.Lines[n]
}

func (b *Buffer) CursorUp() {
	if b.Cursor.Line > 0 {
		b.Cursor.Line--
		b.clampCursor()
	}
}

func (b *Buffer) CursorDown() {
	if b.Cursor.Line < len(b.Lines)-1 {
		b.Cursor.Line++
		b.clampCursor()
	}
}

func (b *Buffer) CursorLeft() {
	if b.Cursor.Col > 0 {
		b.Cursor.Col--
	} else if b.Cursor.Line > 0 {
		b.Cursor.Line--
		b.Cursor.Col = len(b.Lines[b.Cursor.Line])
	}
}

func (b *Buffer) CursorRight() {
	if b.Cursor.Col < len(b.Lines[b.Cursor.Line]) {
		b.Cursor.Col++
	} else if b.Cursor.Line < len(b.Lines)-1 {
		b.Cursor.Line++
		b.Cursor.Col = 0
	}
}

func (b *Buffer) CursorHome() {
	b.Cursor.Col = 0
}

func (b *Buffer) CursorEnd() {
	b.Cursor.Col = len(b.Lines[b.Cursor.Line])
}

func (b *Buffer) CursorWordLeft() {
	if b.Cursor.Col == 0 {
		if b.Cursor.Line > 0 {
			b.Cursor.Line--
			b.Cursor.Col = len(b.Lines[b.Cursor.Line])
		}
		return
	}
	line := b.Lines[b.Cursor.Line]
	col := b.Cursor.Col
	// Skip whitespace/punctuation backwards
	for col > 0 && !isWordChar(rune(line[col-1])) {
		col--
	}
	// Skip word chars backwards
	for col > 0 && isWordChar(rune(line[col-1])) {
		col--
	}
	b.Cursor.Col = col
}

func (b *Buffer) CursorWordRight() {
	line := b.Lines[b.Cursor.Line]
	if b.Cursor.Col >= len(line) {
		if b.Cursor.Line < len(b.Lines)-1 {
			b.Cursor.Line++
			b.Cursor.Col = 0
		}
		return
	}
	col := b.Cursor.Col
	// Skip word chars forward
	for col < len(line) && isWordChar(rune(line[col])) {
		col++
	}
	// Skip whitespace/punctuation forward
	for col < len(line) && !isWordChar(rune(line[col])) {
		col++
	}
	b.Cursor.Col = col
}

func (b *Buffer) PageUp(viewportHeight int) {
	b.Cursor.Line -= viewportHeight
	if b.Cursor.Line < 0 {
		b.Cursor.Line = 0
	}
	b.clampCursor()
}

func (b *Buffer) PageDown(viewportHeight int) {
	b.Cursor.Line += viewportHeight
	if b.Cursor.Line >= len(b.Lines) {
		b.Cursor.Line = len(b.Lines) - 1
	}
	b.clampCursor()
}

func (b *Buffer) Undo() {
	if len(b.undoStack) == 0 {
		return
	}
	edit := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]

	switch edit.Type {
	case EditInsert:
		// Reverse an insert: delete the inserted text
		end := b.advancePos(edit.Pos, edit.Text)
		b.deleteRaw(edit.Pos, end)
		b.Cursor = edit.Pos
	case EditDelete:
		// Reverse a delete: re-insert the deleted text
		b.insertRaw(edit.Pos, edit.Text)
		end := b.advancePos(edit.Pos, edit.Text)
		b.Cursor = end
	}
	b.redoStack = append(b.redoStack, edit)
	b.Dirty = true
}

func (b *Buffer) Redo() {
	if len(b.redoStack) == 0 {
		return
	}
	edit := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]

	switch edit.Type {
	case EditInsert:
		b.insertRaw(edit.Pos, edit.Text)
		b.Cursor = b.advancePos(edit.Pos, edit.Text)
	case EditDelete:
		end := b.advancePos(edit.Pos, edit.Text)
		b.deleteRaw(edit.Pos, end)
		b.Cursor = edit.Pos
	}
	b.undoStack = append(b.undoStack, edit)
	b.Dirty = true
}

func (b *Buffer) Save() error {
	if b.FilePath == "" {
		return nil
	}
	content := strings.Join(b.Lines, "\n") + "\n"
	err := os.WriteFile(b.FilePath, []byte(content), 0644)
	if err == nil {
		b.Dirty = false
	}
	return err
}

func (b *Buffer) SelectedText() string {
	if !b.Selection.Active {
		return ""
	}
	start, end := b.selectionRange()
	return b.textBetween(start, end)
}

func (b *Buffer) DeleteSelection() {
	if !b.Selection.Active {
		return
	}
	start, end := b.selectionRange()
	b.Delete(start, end)
	b.Cursor = start
	b.Selection.Active = false
	b.clampCursor()
}

func (b *Buffer) SelectAll() {
	b.Selection.Active = true
	b.Selection.Anchor = Position{Line: 0, Col: 0}
	last := len(b.Lines) - 1
	b.Selection.Head = Position{Line: last, Col: len(b.Lines[last])}
}

func (b *Buffer) clampCursor() {
	if b.Cursor.Line < 0 {
		b.Cursor.Line = 0
	}
	if b.Cursor.Line >= len(b.Lines) {
		b.Cursor.Line = len(b.Lines) - 1
	}
	if b.Cursor.Col < 0 {
		b.Cursor.Col = 0
	}
	if b.Cursor.Col > len(b.Lines[b.Cursor.Line]) {
		b.Cursor.Col = len(b.Lines[b.Cursor.Line])
	}
}

// --- internal helpers ---

func (b *Buffer) pushUndo(e Edit) {
	b.undoStack = append(b.undoStack, e)
	b.redoStack = nil
}

// insertRaw performs a raw insert without recording undo.
func (b *Buffer) insertRaw(pos Position, text string) {
	line := b.Lines[pos.Line]
	before, after := line[:pos.Col], line[pos.Col:]
	parts := strings.Split(text, "\n")
	if len(parts) == 1 {
		b.Lines[pos.Line] = before + text + after
	} else {
		newLines := make([]string, 0, len(b.Lines)+len(parts)-1)
		newLines = append(newLines, b.Lines[:pos.Line]...)
		newLines = append(newLines, before+parts[0])
		for i := 1; i < len(parts)-1; i++ {
			newLines = append(newLines, parts[i])
		}
		newLines = append(newLines, parts[len(parts)-1]+after)
		newLines = append(newLines, b.Lines[pos.Line+1:]...)
		b.Lines = newLines
	}
}

// deleteRaw performs a raw delete without recording undo.
func (b *Buffer) deleteRaw(start, end Position) {
	before := b.Lines[start.Line][:start.Col]
	after := b.Lines[end.Line][end.Col:]
	b.Lines[start.Line] = before + after
	if end.Line > start.Line {
		b.Lines = append(b.Lines[:start.Line+1], b.Lines[end.Line+1:]...)
	}
}

func (b *Buffer) textBetween(start, end Position) string {
	if start.Line == end.Line {
		return b.Lines[start.Line][start.Col:end.Col]
	}
	var sb strings.Builder
	sb.WriteString(b.Lines[start.Line][start.Col:])
	for i := start.Line + 1; i < end.Line; i++ {
		sb.WriteByte('\n')
		sb.WriteString(b.Lines[i])
	}
	sb.WriteByte('\n')
	sb.WriteString(b.Lines[end.Line][:end.Col])
	return sb.String()
}

func (b *Buffer) advancePos(pos Position, text string) Position {
	for _, ch := range text {
		if ch == '\n' {
			pos.Line++
			pos.Col = 0
		} else {
			pos.Col += len(string(ch))
		}
	}
	return pos
}

func (b *Buffer) selectionRange() (start, end Position) {
	s, e := b.Selection.Anchor, b.Selection.Head
	if e.Line < s.Line || (e.Line == s.Line && e.Col < s.Col) {
		s, e = e, s
	}
	return s, e
}

func leadingWhitespace(s string) string {
	for i, ch := range s {
		if ch != ' ' && ch != '\t' {
			return s[:i]
		}
	}
	return s
}

func isWordChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

// --- Bracket matching ---

var bracketPairs = map[byte]byte{
	'(': ')', '[': ']', '{': '}',
	')': '(', ']': '[', '}': '{',
}
var openBrackets = map[byte]bool{'(': true, '[': true, '{': true}

// MatchBracket returns the position of the matching bracket for the char under cursor.
// Returns (-1,-1) if no bracket at cursor or no match found.
func (b *Buffer) MatchBracket() (int, int) {
	line := b.Lines[b.Cursor.Line]
	if b.Cursor.Col >= len(line) {
		return -1, -1
	}
	ch := line[b.Cursor.Col]
	match, ok := bracketPairs[ch]
	if !ok {
		return -1, -1
	}
	forward := openBrackets[ch]
	depth := 1
	l, c := b.Cursor.Line, b.Cursor.Col

	for {
		if forward {
			c++
		} else {
			c--
		}
		// Wrap lines
		for c < 0 {
			l--
			if l < 0 {
				return -1, -1
			}
			c = len(b.Lines[l]) - 1
		}
		for c >= len(b.Lines[l]) {
			l++
			if l >= len(b.Lines) {
				return -1, -1
			}
			c = 0
		}
		cur := b.Lines[l][c]
		switch cur {
		case ch:
			depth++
		case match:
			depth--
			if depth == 0 {
				return l, c
			}
		}
	}
}

// --- Word boundaries (for double-click) ---

// WordAt returns the start and end column of the word at the given position on a line.
func WordAt(line string, col int) (start, end int) {
	if col >= len(line) {
		return col, col
	}
	ch := rune(line[col])
	if !isWordChar(ch) {
		return col, col + 1
	}
	start = col
	for start > 0 && isWordChar(rune(line[start-1])) {
		start--
	}
	end = col
	for end < len(line) && isWordChar(rune(line[end])) {
		end++
	}
	return start, end
}
