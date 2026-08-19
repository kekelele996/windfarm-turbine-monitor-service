package ops

import (
	"time"

	"windfarm-turbine-monitor-service/internal/report"
)

// ReportRunner generates site reports on a schedule.
type ReportRunner struct {
	Builder *report.Builder
	now     func() time.Time
}

func NewReportRunner(b *report.Builder, now func() time.Time) *ReportRunner {
	if now == nil {
		now = time.Now
	}
	return &ReportRunner{Builder: b, now: now}
}

// LatestSiteReports produces a report for every known site over the last day.
func (r *ReportRunner) LatestSiteReports() map[string]report.Report {
	out := map[string]report.Report{}
	now := r.now()
	for _, t := range r.Builder.Turbines().List() {
		if _, done := out[t.Site]; done {
			continue
		}
		out[t.Site] = r.Builder.Build(t.Site, now.Add(-24*time.Hour), now)
	}
	return out
}
