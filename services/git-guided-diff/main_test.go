package gitguideddiff

import "testing"

func TestPlanSplitsLargeReview(t *testing.T) {
	steps := Plan([]FileDiff{{"a.go", 3, 1, ""}, {"b.go", 4, 2, ""}, {"c.go", 1, 0, ""}}, 7)
	if len(steps) != 2 || len(steps[0].Files) != 1 || len(steps[1].Files) != 2 || steps[1].Number != 2 {
		t.Fatalf("%+v", steps)
	}
}
