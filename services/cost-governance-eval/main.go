package costgovernanceeval

import "sort"

type Resource struct {
	Provider, ID, Kind string
	MonthlyCost        float64
	Utilization        float64
	AIWorkload         bool
}
type Finding struct {
	Resource Resource
	Reason   string
	Savings  float64
}
type Report struct {
	Findings     []Finding
	TotalSavings float64
}

func Evaluate(resources []Resource) Report {
	report := Report{}
	for _, resource := range resources {
		if resource.Utilization < 0.2 {
			rate := 0.7
			reason := "underutilized infrastructure"
			if resource.AIWorkload {
				rate = 0.5
				reason = "underutilized AI workload"
			}
			saving := resource.MonthlyCost * rate
			report.Findings = append(report.Findings, Finding{resource, reason, saving})
			report.TotalSavings += saving
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool { return report.Findings[i].Savings > report.Findings[j].Savings })
	return report
}
