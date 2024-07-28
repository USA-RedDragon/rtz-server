package utils

import "math"

const earthRadiusMeters = 6371000

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// Haversine returns the distance between two GPS coordinates in meters.
func Haversine(startLat, startLng, endLat, endLng float64) float64 {
	phi1 := degToRad(startLat)
	phi2 := degToRad(endLat)
	deltaPhi := degToRad(endLat - startLat)
	deltaLambda := degToRad(endLng - startLng)

	a := math.Pow(math.Sin(deltaPhi/2), 2) + math.Cos(phi1)*math.Cos(phi2)*
		math.Pow(math.Sin(deltaLambda/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
