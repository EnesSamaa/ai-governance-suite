package costgovernanceeval

import "testing"

func TestEvaluateFindsIdleSpend(t *testing.T) {
	report := Evaluate([]Resource{{"AWS", "i-1", "vm", 100, 0.1, false}, {"Azure", "ai-1", "gpu", 200, 0.1, true}, {"AWS", "busy", "db", 300, 0.9, false}})
	if len(report.Findings) != 2 || report.TotalSavings != 170 {
		t.Fatalf("%+v", report)
	}
}
