package report

import (
	"fmt"
	"time"

	"windfarm-turbine-monitor-service/internal/analytics"
	"windfarm-turbine-monitor-service/internal/platform"
	"windfarm-turbine-monitor-service/internal/telemetry"
	"windfarm-turbine-monitor-service/internal/turbine"
)

// Builder assembles site reports from fleet and telemetry data.
type Builder struct {
	registry *turbine.Registry
	samples  *telemetry.SampleRegistry
	now      func() time.Time
}

func NewBuilder(registry *turbine.Registry, samples *telemetry.SampleRegistry, now func() time.Time) *Builder {
	if now == nil {
		now = time.Now
	}
	return &Builder{registry: registry, samples: samples, now: now}
}

// Build produces a report for a site over the given period.
func (b *Builder) Build(site string, start, end time.Time) Report {
	rep := Report{
		ID:          platform.NewID("rep"),
		Site:        site,
		PeriodStart: start,
		PeriodEnd:   end,
	}
	fleet := b.registry.Stats()
	rep.Sections = append(rep.Sections, Section{
		Title: "Fleet",
		Rows: []SectionRow{
			{Key: "total", Value: fmt.Sprintf("%d", fleet.Total)},
			{Key: "active", Value: fmt.Sprintf("%d", fleet.Active)},
			{Key: "maintenance", Value: fmt.Sprintf("%d", fleet.Maintenance)},
		},
	})
	rep.Sections = append(rep.Sections, b.energySection(site, start, end))
	return rep
}

func (b *Builder) energySection(site string, start, end time.Time) Section {
	section := Section{Title: "Energy"}
	for _, t := range b.registry.List() {
		if t.Site != site {
			continue
		}
		samples := b.samples.Since(t.ID, start)
		var within []telemetry.Sample
		for _, s := range samples {
			if !s.Timestamp.After(end) {
				within = append(within, s)
			}
		}
		agg := analytics.AggregateSamples(within, end.Sub(start).Seconds())
		section.Rows = append(section.Rows, SectionRow{
			Key:   t.ID,
			Value: fmt.Sprintf("%.2f kWh", agg.EnergyKWh),
		})
	}
	return section
}

// Turbines exposes the fleet registry backing this builder.
func (b *Builder) Turbines() *turbine.Registry { return b.registry }
