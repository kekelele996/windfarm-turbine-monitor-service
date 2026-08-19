package alarm

// Route classifies an alarm severity into a routing tier.
type Route int

const (
	RouteSilent Route = iota
	RouteOps
	RouteEngineer
	RouteCritical
)

func RouteForSeverity(severity string) Route {
	switch severity {
	case "critical":
		return RouteCritical
	case "warning":
		return RouteEngineer
	case "info":
		return RouteOps
	default:
		return RouteSilent
	}
}

// Router selects notifier channels based on severity.
type Router struct {
	channels map[Route][]chan Alarm
}

func NewRouter() *Router { return &Router{channels: map[Route][]chan Alarm{}} }

func (r *Router) Register(route Route, ch chan Alarm) {
	r.channels[route] = append(r.channels[route], ch)
}

func (r *Router) Fanout(a Alarm) int {
	route := RouteForSeverity(a.Severity)
	sent := 0
	for _, ch := range r.channels[route] {
		select {
		case ch <- a:
			sent++
		default:
		}
	}
	return sent
}
