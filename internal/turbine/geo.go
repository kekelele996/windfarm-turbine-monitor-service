package turbine

import "math"

// Position is a WGS-84 coordinate for a turbine.
type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// DistanceMeters computes the great-circle distance between two positions.
func DistanceMeters(a, b Position) float64 {
	const earthRadius = 6371000.0
	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180
	dLat := (b.Latitude - a.Latitude) * math.Pi / 180
	dLon := (b.Longitude - a.Longitude) * math.Pi / 180
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * earthRadius * math.Asin(math.Sqrt(h))
}

// ClosestSite finds the nearest named site to a position.
type SiteLocation struct {
	Name string
	Pos  Position
}

func ClosestSite(pos Position, sites []SiteLocation) string {
	if len(sites) == 0 {
		return ""
	}
	best := sites[0]
	bestDist := DistanceMeters(pos, best.Pos)
	for _, s := range sites[1:] {
		d := DistanceMeters(pos, s.Pos)
		if d < bestDist {
			best = s
			bestDist = d
		}
	}
	return best.Name
}
