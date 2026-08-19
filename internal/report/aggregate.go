package report

// Totals rolls up section rows into a flat string map.
func (r Report) Totals() map[string]string {
	out := map[string]string{}
	for _, sec := range r.Sections {
		for _, row := range sec.Rows {
			out[sec.Title+"."+row.Key] = row.Value
		}
	}
	return out
}
