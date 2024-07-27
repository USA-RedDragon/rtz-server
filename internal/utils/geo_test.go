package utils_test

import (
	"math"
	"testing"

	"github.com/USA-RedDragon/rtz-server/internal/utils"
)

type coords struct {
	lat float64
	lng float64
}

var (
	devonTower      = coords{35.4669626, -97.5280147}
	anthemBrewing   = coords{35.4674537, -97.5331325}
	willRogers      = coords{35.3954731, -97.6065239}
	ouCampus        = coords{35.3956022, -97.9258855}
	rocklahoma      = coords{36.3638353, -95.2886689}
	gatewayArch     = coords{38.6251432, -90.1970501}
	statueOfLiberty = coords{40.6892494, -74.0445004}
	reykjavik       = coords{64.1334904, -21.8524423}
	tokyo           = coords{35.5092405, 139.7698121}
)

func TestHaversine(t *testing.T) {
	t.Parallel()

	// Short distance: devon tower to anthem brewing
	dist := math.Round(utils.Haversine(devonTower.lat, devonTower.lng, anthemBrewing.lat, anthemBrewing.lng))
	if dist != 467 {
		t.Errorf("expected 467 meters between Devon Tower and Anthem Brewing, got %f", dist)
	}

	// Reverse short distance: anthem brewing to devon tower
	dist = math.Round(utils.Haversine(anthemBrewing.lat, anthemBrewing.lng, devonTower.lat, devonTower.lng))
	if dist != 467 {
		t.Errorf("expected 467 meters between Anthem Brewing and Devon Tower, got %f", dist)
	}

	// Medium distance: devon tower to willRogers
	dist = math.Round(utils.Haversine(devonTower.lat, devonTower.lng, willRogers.lat, willRogers.lng))
	if dist != 10667 {
		t.Errorf("expected 10667 meters between Devon Tower and Will Rogers, got %f", dist)
	}

	// Reverse medium distance: willRogers to devon tower
	dist = math.Round(utils.Haversine(willRogers.lat, willRogers.lng, devonTower.lat, devonTower.lng))
	if dist != 10667 {
		t.Errorf("expected 10667 meters between Will Rogers and Devon Tower, got %f", dist)
	}

	// Medium distance: ouCampus to rocklahoma
	dist = math.Round(utils.Haversine(ouCampus.lat, ouCampus.lng, rocklahoma.lat, rocklahoma.lng))
	if dist != 260843 {
		t.Errorf("expected 260843 meters between OU Campus and Rocklahoma, got %f", dist)
	}

	// Reverse medium distance: rocklahoma to ouCampus
	dist = math.Round(utils.Haversine(rocklahoma.lat, rocklahoma.lng, ouCampus.lat, ouCampus.lng))
	if dist != 260843 {
		t.Errorf("expected 260843 meters between Rocklahoma and OU Campus, got %f", dist)
	}

	// Long distance: gatewayArch to statueOfLiberty
	dist = math.Round(utils.Haversine(gatewayArch.lat, gatewayArch.lng, statueOfLiberty.lat, statueOfLiberty.lng))
	if dist != 1399606 {
		t.Errorf("expected 1399606 meters between Gateway Arch and Statue of Liberty, got %f", dist)
	}

	// Reverse long distance: statueOfLiberty to gatewayArch
	dist = math.Round(utils.Haversine(statueOfLiberty.lat, statueOfLiberty.lng, gatewayArch.lat, gatewayArch.lng))
	if dist != 1399606 {
		t.Errorf("expected 1399606 meters between Statue of Liberty and Gateway Arch, got %f", dist)
	}

	// Very long distance: reykjavík to tokyo
	dist = math.Round(utils.Haversine(reykjavik.lat, reykjavik.lng, tokyo.lat, tokyo.lng))
	if dist != 8818082 {
		t.Errorf("expected 8818082 meters between Reykjavík and Tokyo, got %f", dist)
	}

	// Reverse very long distance: tokyo to reykjavík
	dist = math.Round(utils.Haversine(tokyo.lat, tokyo.lng, reykjavik.lat, reykjavik.lng))
	if dist != 8818082 {
		t.Errorf("expected 8818082 meters between Tokyo and Reykjavík, got %f", dist)
	}

	// Very long distance: reykjavík to gatewayArch
	dist = math.Round(utils.Haversine(reykjavik.lat, reykjavik.lng, gatewayArch.lat, gatewayArch.lng))
	if dist != 5178408 {
		t.Errorf("expected 5178408 meters between Reykjavík and Gateway Arch, got %f", dist)
	}

	// Reverse very long distance: gatewayArch to reykjavík
	dist = math.Round(utils.Haversine(gatewayArch.lat, gatewayArch.lng, reykjavik.lat, reykjavik.lng))
	if dist != 5178408 {
		t.Errorf("expected 5178408 meters between Gateway Arch and Reykjavík, got %f", dist)
	}

	// Very long distance: tokyo to statueOfLiberty
	dist = math.Round(utils.Haversine(tokyo.lat, tokyo.lng, statueOfLiberty.lat, statueOfLiberty.lng))
	if dist != 10864801 {
		t.Errorf("expected 10864801 meters between Tokyo and Statue of Liberty, got %f", dist)
	}

	// Reverse very long distance: statueOfLiberty to tokyo
	dist = math.Round(utils.Haversine(statueOfLiberty.lat, statueOfLiberty.lng, tokyo.lat, tokyo.lng))
	if dist != 10864801 {
		t.Errorf("expected 10864801 meters between Statue of Liberty and Tokyo, got %f", dist)
	}
}
