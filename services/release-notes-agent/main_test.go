package releasenotesagent

import (
	"strings"
	"testing"
)

func TestGenerateSemanticRelease(t *testing.T) {
	release := Generate("v1.2.0", []Commit{{Subject: "fix: null panic"}, {Subject: "feat: export audit"}, {Subject: "feat!: remove legacy"}})
	if len(release.Features) != 1 || len(release.Breaking) != 1 || !strings.Contains(release.Markdown(), "Breaking") {
		t.Fatalf("%+v", release)
	}
}
