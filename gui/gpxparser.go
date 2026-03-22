package main

import (
	"encoding/xml"
	"math"
	"os"
)

// TrackPoint is a single lat/lon/elevation from a GPX track.
type TrackPoint struct {
	Lat float64
	Lon float64
	Ele float64 // math.NaN() if no elevation data
}

type gpxXML struct {
	XMLName xml.Name   `xml:"gpx"`
	Tracks  []gpxTrack `xml:"trk"`
}

type gpxTrack struct {
	Segments []gpxSegment `xml:"trkseg"`
}

type gpxSegment struct {
	Points []gpxPoint `xml:"trkpt"`
}

type gpxPoint struct {
	Lat float64  `xml:"lat,attr"`
	Lon float64  `xml:"lon,attr"`
	Ele *float64 `xml:"ele"`
}

// ParseGPX reads a GPX file and returns all track points.
// Points with no <ele> element have Ele = math.NaN().
func ParseGPX(path string) ([]TrackPoint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var g gpxXML
	dec := xml.NewDecoder(f)
	if err := dec.Decode(&g); err != nil {
		return nil, err
	}

	var points []TrackPoint
	for _, trk := range g.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				tp := TrackPoint{Lat: pt.Lat, Lon: pt.Lon, Ele: math.NaN()}
				if pt.Ele != nil {
					tp.Ele = *pt.Ele
				}
				points = append(points, tp)
			}
		}
	}
	return points, nil
}

// CumulativeDistance returns per-point cumulative distance in metres,
// parallel to pts. Uses the haversine formula.
func CumulativeDistance(pts []TrackPoint) []float64 {
	dist := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		dist[i] = dist[i-1] + haversine(pts[i-1], pts[i])
	}
	return dist
}

func haversine(a, b TrackPoint) float64 {
	const R = 6371000.0
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	aa := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return R * 2 * math.Atan2(math.Sqrt(aa), math.Sqrt(1-aa))
}
