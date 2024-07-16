package logparser

import (
	"fmt"
	"io"
	"math"

	"capnproto.org/go/capnp/v3"
	"github.com/USA-RedDragon/rtz-server/internal/cereal"
)

type GpsCoordinates struct {
	Latitude             float64
	Longitude            float64
	LogMonoTime          uint64
	AccuracyMeters       float64
	SpeedMetersPerSecond float64
	Bearing              float64
	Distance             float64
}

type SegmentData struct {
	GPSLocations            []GpsCoordinates
	EndCoordinates          GpsCoordinates
	EndLogMonoTime          uint64
	FirstClockWallTimeNanos uint64
	FirstClockLogMonoTime   uint64
	CANPresent              bool
	GitDirty                bool
	GitCommit               string
	Version                 string
	DongleID                string
	Radar                   bool
	InitLogMonoTime         uint64
	DeviceType              cereal.InitData_DeviceType
	CarModel                string
	GitRemote               string
	GitBranch               string
	StartOfRoute            bool
	EndOfRoute              bool
}

const earthRadiusMeters = 6371000

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// haversine returns the distance between two GPS coordinates in meters.
func haversine(endLat, endLng, startLat, startLng float64) float64 {
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

func DecodeSegmentData(reader io.Reader) (SegmentData, error) {
	var segmentData SegmentData

	decoder := capnp.NewDecoder(reader)
	gpsCnt := 0
	for {
		msg, err := decoder.Decode()
		if err != nil {
			if err != io.EOF {
				return SegmentData{}, fmt.Errorf("failed to decode log: %w", err)
			}
			break
		}
		event, err := cereal.ReadRootEvent(msg)
		if err != nil {
			return SegmentData{}, fmt.Errorf("failed to read event: %w", err)
		}
		segmentData.EndLogMonoTime = event.LogMonoTime()
		// We're definitely not going to be handling every event type, so we can ignore the exhaustive linter warning
		//nolint:golint,exhaustive
		switch event.Which() {
		case cereal.Event_Which_can:
			segmentData.CANPresent = true
		case cereal.Event_Which_gpsLocation:
			gpsLocation, err := event.GpsLocation()
			if err != nil {
				return SegmentData{}, err
			}
			gps := GpsCoordinates{
				Latitude:             gpsLocation.Latitude(),
				Longitude:            gpsLocation.Longitude(),
				AccuracyMeters:       float64(gpsLocation.HorizontalAccuracy()),
				SpeedMetersPerSecond: float64(gpsLocation.Speed()),
				Bearing:              float64(gpsLocation.BearingDeg()),
				LogMonoTime:          event.LogMonoTime(),
			}
			// Sample only evert 100th GPS point
			if gpsCnt%100 == 0 {
				segmentData.GPSLocations = append(segmentData.GPSLocations, gps)
			}
			segmentData.EndCoordinates = gps
		case cereal.Event_Which_sentinel:
			sentinel, err := event.Sentinel()
			if err != nil {
				return SegmentData{}, err
			}
			switch sentinel.Type() {
			case cereal.Sentinel_SentinelType_startOfRoute:
				segmentData.StartOfRoute = true
			case cereal.Sentinel_SentinelType_endOfRoute:
				segmentData.EndOfRoute = true
			}
		case cereal.Event_Which_radarState:
			segmentData.Radar = true
		case cereal.Event_Which_clocks:
			clocks, err := event.Clocks()
			if err != nil {
				return SegmentData{}, err
			}
			time := clocks.WallTimeNanos()
			if segmentData.FirstClockWallTimeNanos == 0 {
				segmentData.FirstClockWallTimeNanos = time
				segmentData.FirstClockLogMonoTime = event.LogMonoTime()
			}
		case cereal.Event_Which_initData:
			initData, err := event.InitData()
			if err != nil {
				return SegmentData{}, err
			}
			remote, err := initData.GitRemote()
			if err != nil {
				return SegmentData{}, err
			}
			segmentData.GitRemote = remote
			branch, err := initData.GitBranch()
			if err != nil {
				return SegmentData{}, err
			}
			segmentData.GitBranch = branch

			segmentData.InitLogMonoTime = event.LogMonoTime()

			segmentData.GitDirty = initData.Dirty()
			commit, err := initData.GitCommit()
			if err != nil {
				return SegmentData{}, err
			}
			segmentData.GitCommit = commit
			vers, err := initData.Version()
			if err != nil {
				return SegmentData{}, err
			}
			segmentData.Version = vers

			segmentData.DeviceType = initData.DeviceType()

			segmentData.DongleID, err = initData.DongleId()
			if err != nil {
				return SegmentData{}, err
			}

			paramProto, err := initData.Params()
			if err != nil {
				return SegmentData{}, err
			}
			params, err := paramProto.Entries()
			if err != nil {
				return SegmentData{}, err
			}
			for i := 0; i < params.Len(); i++ {
				param := params.At(i)
				keyPtr, err := param.Key()
				if err != nil {
					return SegmentData{}, err
				}
				valPtr, err := param.Value()
				if err != nil {
					return SegmentData{}, err
				}
				key := keyPtr.Text()
				val := valPtr.Data()
				if key == "CarModel" {
					segmentData.CarModel = string(val)
				}
			}
		}
	}

	return segmentData, nil
}
