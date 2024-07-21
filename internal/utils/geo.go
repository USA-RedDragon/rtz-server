package utils

import "math"

const earthRadiusMeters = 6371000

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// Haversine returns the distance between two GPS coordinates in meters.
func Haversine(startLat, startLng, endLat, endLng float64) float64 {
	endLatRads := degToRad(endLat)
	endLngRads := degToRad(endLng)
	startLatRads := degToRad(startLat)
	startLngRads := degToRad(startLng)

	deltaLat := math.Abs(endLatRads - startLatRads)
	deltaLng := math.Abs(endLngRads - startLngRads)

	a := math.Pow(math.Sin(deltaLat/2), 2) + math.Cos(startLatRads)*math.Cos(endLatRads)*math.Pow(math.Sin(deltaLng/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
