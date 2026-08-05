package releasenotesagent

import (
	"sort"
	"strings"
)

type Commit struct{ SHA, Subject, PR string }
type Release struct {
	Version                   string
	Features, Fixes, Breaking []string
}

func Generate(version string, commits []Commit) Release {
	release := Release{Version: version}
	for _, commit := range commits {
		text := strings.TrimSpace(commit.Subject)
		lower := strings.ToLower(text)
		switch {
		case strings.Contains(lower, "breaking") || strings.HasPrefix(lower, "feat!"):
			release.Breaking = append(release.Breaking, text)
		case strings.HasPrefix(lower, "feat"):
			release.Features = append(release.Features, text)
		case strings.HasPrefix(lower, "fix"):
			release.Fixes = append(release.Fixes, text)
		}
	}
	sort.Strings(release.Features)
	sort.Strings(release.Fixes)
	sort.Strings(release.Breaking)
	return release
}
func (r Release) Markdown() string {
	sections := []string{"# " + r.Version}
	if len(r.Features) > 0 {
		sections = append(sections, "## Features\n- "+strings.Join(r.Features, "\n- "))
	}
	if len(r.Fixes) > 0 {
		sections = append(sections, "## Fixes\n- "+strings.Join(r.Fixes, "\n- "))
	}
	if len(r.Breaking) > 0 {
		sections = append(sections, "## Breaking changes\n- "+strings.Join(r.Breaking, "\n- "))
	}
	return strings.Join(sections, "\n\n")
}
