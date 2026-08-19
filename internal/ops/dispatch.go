package ops

import (
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
)

// AlarmDispatch routes alarms through notifier channels and applies throttling
// and escalation timers.
type AlarmDispatch struct {
	Dispatcher *alarm.Dispatcher
	Router     *alarm.Router
	Throttle   *alarm.Throttle
	Audit      *audit.Stream
	mu         sync.Mutex
}

func NewAlarmDispatch(d *alarm.Dispatcher, r *alarm.Router, t *alarm.Throttle, stream *audit.Stream) *AlarmDispatch {
	return &AlarmDispatch{Dispatcher: d, Router: r, Throttle: t, Audit: stream}
}

// DispatchAndRoute applies throttle, dispatches and routes the alarm.
func (a *AlarmDispatch) DispatchAndRoute(candidate alarm.Alarm, now time.Time) (alarm.Alarm, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.Throttle.Allow(candidate.TurbineID, now) {
		return alarm.Alarm{}, false
	}
	created, ok, _ := a.Dispatcher.Dispatch(candidate)
	if !ok {
		return alarm.Alarm{}, false
	}
	if a.Router != nil {
		a.Router.Fanout(created)
	}
	if a.Audit != nil {
		a.Audit.Publish(audit.EventAlarm, created.TurbineID, created.RuleID)
	}
	return created, true
}

// Escalate moves long-open alarms to a higher routing tier.
func (a *AlarmDispatch) Escalate(age time.Duration) int {
	count := 0
	for _, al := range a.Dispatcher.ListOpen() {
		if time.Since(al.OccurredAt) > age {
			_, _ = a.Dispatcher.Ack(al.ID)
			count++
		}
	}
	return count
}
