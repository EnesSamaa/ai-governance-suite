package gitguideddiff

import (
	"sort"
	"strings"
)

type FileDiff struct {
	Path           string
	Added, Removed int
	Patch          string
}
type ReviewStep struct {
	Number  int
	Files   []FileDiff
	Summary string
}

func Plan(files []FileDiff, maxChanges int) []ReviewStep {
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	var steps []ReviewStep
	var current ReviewStep
	changes := 0
	for _, file := range files {
		size := file.Added + file.Removed
		if len(current.Files) > 0 && changes+size > maxChanges {
			current.Number = len(steps) + 1
			current.Summary = summary(current.Files)
			steps = append(steps, current)
			current = ReviewStep{}
			changes = 0
		}
		current.Files = append(current.Files, file)
		changes += size
	}
	if len(current.Files) > 0 {
		current.Number = len(steps) + 1
		current.Summary = summary(current.Files)
		steps = append(steps, current)
	}
	return steps
}
func summary(files []FileDiff) string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.Path
	}
	return "Review " + strings.Join(paths, ", ")
}
