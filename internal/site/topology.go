package site

// Topology describes the layout of turbines within a site.
type Topology struct {
	Site     string
	Clusters map[string][]string // cluster id -> turbine ids
}

func NewTopology(site string) *Topology {
	return &Topology{Site: site, Clusters: map[string][]string{}}
}

func (t *Topology) Assign(cluster string, turbineID string) {
	t.Clusters[cluster] = append(t.Clusters[cluster], turbineID)
}

// ClusterOf returns the cluster containing a turbine.
func (t *Topology) ClusterOf(turbineID string) string {
	for cluster, ids := range t.Clusters {
		for _, id := range ids {
			if id == turbineID {
				return cluster
			}
		}
	}
	return ""
}

// ClusterSizes returns a map of cluster id to turbine count.
func (t *Topology) ClusterSizes() map[string]int {
	out := map[string]int{}
	for c, ids := range t.Clusters {
		out[c] = len(ids)
	}
	return out
}
