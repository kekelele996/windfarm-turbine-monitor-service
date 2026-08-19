package report

import "time"

// Report is a rolled-up operational summary for a site.
type Report struct {
	ID          string    `json:"id"`
	Site        string    `json:"site"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Sections    []Section `json:"sections"`
}

type Section struct {
	Title string       `json:"title"`
	Rows  []SectionRow `json:"rows"`
}

type SectionRow struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
