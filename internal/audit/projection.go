package audit

import "sort"

// Projection aggregates events by type and turbine for dashboards.
type Projection struct {
	ByType    map[EventType]int
	ByTurbine map[string]int
}

func Project(events []Event) Projection {
	p := Projection{ByType: map[EventType]int{}, ByTurbine: map[string]int{}}
	for _, e := range events {
		p.ByType[e.Type]++
		p.ByTurbine[e.TurbineID]++
	}
	return p
}

// TopTurbines returns turbine IDs ordered by descending event count.
func (p Projection) TopTurbines(limit int) []string {
	type kv struct {
		id string
		n  int
	}
	list := make([]kv, 0, len(p.ByTurbine))
	for id, n := range p.ByTurbine {
		list = append(list, kv{id, n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n != list[j].n {
			return list[i].n > list[j].n
		}
		return list[i].id < list[j].id
	})
	out := make([]string, 0, limit)
	for i, item := range list {
		if i >= limit {
			break
		}
		out = append(out, item.id)
	}
	return out
}
