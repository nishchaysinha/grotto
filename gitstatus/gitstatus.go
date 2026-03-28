package gitstatus

import (
	"bufio"
	"bytes"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FileStatus represents the git status of a file.
type FileStatus int

const (
	StatusUnchanged FileStatus = iota
	StatusModified
	StatusAdded
	StatusUntracked
	StatusDeleted
	StatusRenamed
)

// LineChange represents the type of change on a specific line.
type LineChange int

const (
	LineUnchanged LineChange = iota
	LineAdded
	LineModified
	LineDeleted // marker placed at the line adjacent to a pure deletion
)

// GetFileStatuses runs git status --porcelain in repoRoot and returns a map
// of absolute file paths to their FileStatus.
func GetFileStatuses(repoRoot string) map[string]FileStatus {
	cmd := exec.Command("git", "-C", repoRoot, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	result := make(map[string]FileStatus)
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if len(line) < 4 {
			continue
		}
		x := line[0] // index/staged status
		y := line[1] // worktree/unstaged status
		path := strings.TrimSpace(line[3:])
		// Handle renames: "old -> new"
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		absPath := filepath.Join(repoRoot, path)
		switch {
		case x == '?' && y == '?':
			result[absPath] = StatusUntracked
		case x == 'A':
			result[absPath] = StatusAdded
		case x == 'D' || y == 'D':
			result[absPath] = StatusDeleted
		case x == 'R':
			result[absPath] = StatusRenamed
		default:
			result[absPath] = StatusModified
		}
	}
	return result
}

// GetLineDiff returns per-line change status for a file, keyed by 0-based line number.
// Tries git diff HEAD first, then git diff --cached for staged-only new files.
func GetLineDiff(repoRoot, filePath string) map[int]LineChange {
	result := make(map[int]LineChange)
	out := runDiff(repoRoot, "HEAD", filePath)
	if len(out) == 0 {
		out = runDiff(repoRoot, "--cached", filePath)
	}
	if len(out) == 0 {
		return result
	}
	parseDiff(out, result)
	return result
}

// FindRoot returns the git repository root for the given path,
// or empty string if not in a git repo.
func FindRoot(path string) string {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runDiff(repoRoot, ref, filePath string) []byte {
	var args []string
	if ref == "--cached" {
		args = []string{"-C", repoRoot, "diff", "--cached", "--unified=0", "--", filePath}
	} else {
		args = []string{"-C", repoRoot, "diff", ref, "--unified=0", "--", filePath}
	}
	out, _ := exec.Command("git", args...).Output()
	return out
}

func parseDiff(out []byte, result map[int]LineChange) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	newLine := 0
	var addLines []int
	hasDeletes := false

	flush := func() {
		if len(addLines) > 0 {
			change := LineAdded
			if hasDeletes {
				change = LineModified
			}
			for _, l := range addLines {
				result[l] = change
			}
		} else if hasDeletes {
			// Pure deletion: mark the adjacent line (newLine is the next context line)
			if _, exists := result[newLine]; !exists {
				result[newLine] = LineDeleted
			}
		}
		addLines = addLines[:0]
		hasDeletes = false
	}

	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "@@"):
			flush()
			newLine = parseHunkStart(line)
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			addLines = append(addLines, newLine)
			newLine++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			hasDeletes = true
		case len(line) > 0 && line[0] == ' ':
			flush()
			newLine++
		}
	}
	flush()
}

// parseHunkStart extracts the new-file start line (0-based) from a @@ header.
func parseHunkStart(line string) int {
	// Format: @@ -a[,b] +c[,d] @@ ...
	i := strings.Index(line, "+")
	if i < 0 {
		return 0
	}
	rest := line[i+1:]
	j := strings.IndexAny(rest, ", @")
	if j >= 0 {
		rest = rest[:j]
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 1 {
		return 0
	}
	return n - 1 // convert to 0-indexed
}
