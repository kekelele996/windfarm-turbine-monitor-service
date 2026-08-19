package grid

// ConnectionState describes the grid interconnection status.
type ConnectionState string

const (
	Connected    ConnectionState = "connected"
	Islanded     ConnectionState = "islanded"
	Disconnected ConnectionState = "disconnected"
)

// Connection tracks grid link health per site.
type Connection struct {
	Site    string
	State   ConnectionState
	Voltage float64
}

// AllowExport reports whether power may be exported under the current state.
func AllowExport(c Connection) bool {
	return c.State == Connected && c.Voltage >= 0.9
}

// ConnRegistry stores connection state by site.
type ConnRegistry struct {
	conns map[string]Connection
}

func NewConnRegistry() *ConnRegistry { return &ConnRegistry{conns: map[string]Connection{}} }

func (r *ConnRegistry) Set(c Connection) { r.conns[c.Site] = c }

func (r *ConnRegistry) Get(site string) (Connection, bool) {
	c, ok := r.conns[site]
	return c, ok
}
