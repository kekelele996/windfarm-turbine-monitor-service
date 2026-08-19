package report

// MergeSections combines two sections by title, appending rows.
func MergeSections(a, b Section) Section {
	if a.Title == "" {
		return b
	}
	if a.Title != b.Title {
		return a
	}
	merged := a
	merged.Rows = append(merged.Rows, b.Rows...)
	return merged
}

// RowMap converts a section's rows into a lookup map.
func (s Section) RowMap() map[string]string {
	out := make(map[string]string, len(s.Rows))
	for _, r := range s.Rows {
		out[r.Key] = r.Value
	}
	return out
}

// cloneRows deep-copies a section row slice before sorting.
func cloneRows(in []SectionRow) []SectionRow {
	out := make([]SectionRow, len(in))
	copy(out, in)
	return out
}

// SortRows orders section rows by key.
func SortRows(s Section) Section {
	rows := cloneRows(s.Rows)
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].Key < rows[j-1].Key; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	s.Rows = rows
	return s
}
