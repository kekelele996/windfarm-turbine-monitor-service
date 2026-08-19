package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/gateway"
	"windfarm-turbine-monitor-service/internal/report"
	"windfarm-turbine-monitor-service/internal/ruleengine"
	"windfarm-turbine-monitor-service/internal/telemetry"
	"windfarm-turbine-monitor-service/internal/turbine"
	"windfarm-turbine-monitor-service/internal/workorder"
)

func main() {
	cfg := turbine.FromEnv()

	seed := []turbine.Turbine{
		{ID: "WTG-001", Name: "Turbine Alpha", Model: "V90", Site: cfg.Site, RatedPowerKW: 2000, CutInWindMPS: cfg.DefaultCutIn, CutOutWindMPS: cfg.DefaultCutOut, State: turbine.StateRunning, Commissioned: time.Now().Add(-700 * 24 * time.Hour), Firmware: "fw-1.4.2"},
		{ID: "WTG-002", Name: "Turbine Beta", Model: "V112", Site: cfg.Site, RatedPowerKW: 3000, CutInWindMPS: cfg.DefaultCutIn, CutOutWindMPS: cfg.DefaultCutOut, State: turbine.StateRunning, Commissioned: time.Now().Add(-400 * 24 * time.Hour), Firmware: "fw-1.4.2"},
		{ID: "WTG-003", Name: "Turbine Gamma", Model: "V90", Site: cfg.Site, RatedPowerKW: 2000, CutInWindMPS: cfg.DefaultCutIn, CutOutWindMPS: cfg.DefaultCutOut, State: turbine.StateMaintenance, Commissioned: time.Now().Add(-900 * 24 * time.Hour), Firmware: "fw-1.3.9"},
	}
	registry := turbine.NewRegistry(seed)

	samples := telemetry.NewSampleRegistry(1024)
	ingestor := telemetry.NewIngestor(samples, 4)

	rules := ruleengine.NewRegistry(nil)
	for _, t := range registry.List() {
		for _, rule := range ruleengine.ThresholdSet(t, cfg) {
			_ = rules.Put(rule)
		}
	}
	evaluator := ruleengine.NewEvaluator(rules)

	faultStore := fault.NewStore()
	faultSvc := fault.NewService(faultStore)

	dedup := alarm.NewDeduplicator(5 * time.Minute)
	dispatcher := alarm.NewDispatcher(dedup)
	notifier := alarm.NewNotifier()

	crews := workorder.NewCrewPool([]string{"crew-a", "crew-b", "crew-c"})
	orders := workorder.NewScheduler(crews, nil)

	stream := audit.NewStream()
	builder := report.NewBuilder(registry, samples, nil)

	app := &gateway.App{
		Turbines:  registry,
		Samples:   samples,
		Ingestor:  ingestor,
		Rules:     rules,
		Evaluator: evaluator,
		Faults:    faultSvc,
		Alarms:    dispatcher,
		Notifier:  notifier,
		Orders:    orders,
		Audit:     stream,
		Reports:   builder,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "18080"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           gateway.NewRouter(app),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	log.Printf("windfarm turbine monitor listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
