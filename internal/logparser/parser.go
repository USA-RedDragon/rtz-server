package logparser

import (
	"fmt"
	"io"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/USA-RedDragon/rtz-server/internal/cereal"
)

type GpsCoordinates struct {
	Latitude             float64
	Longitude            float64
	Time                 time.Time
	AccuracyMeters       float64
	SpeedMetersPerSecond float64
	Bearing              float64
}

type SegmentData struct {
	GPSLocations      []GpsCoordinates
	EndCoordinates    GpsCoordinates
	EarliestTimestamp uint64
	LatestTimestamp   uint64
	CANPresent        bool
	DongleID          string
	InitLogMonoTime   uint64
	DeviceType        cereal.InitData_DeviceType
	CarModel          string
	GitRemote         string
	GitBranch         string
	StartOfRoute      bool
	EndOfRoute        bool
}

func DecodeSegmentData(reader io.Reader) (SegmentData, error) {
	var segmentData SegmentData

	decoder := capnp.NewDecoder(reader)
	cnt := 0
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
		if cnt < 5 {
			fmt.Println(event.Which())
			cnt++
		}
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
			// TODO: associate logMonoTime with a wall time
			gps := GpsCoordinates{
				Latitude:             gpsLocation.Latitude(),
				Longitude:            gpsLocation.Longitude(),
				AccuracyMeters:       float64(gpsLocation.HorizontalAccuracy()),
				SpeedMetersPerSecond: float64(gpsLocation.Speed()),
				Bearing:              float64(gpsLocation.BearingDeg()),
			}
			segmentData.GPSLocations = append(segmentData.GPSLocations, gps)
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
		case cereal.Event_Which_clocks:
			clocks, err := event.Clocks()
			if err != nil {
				return SegmentData{}, err
			}
			time := clocks.WallTimeNanos()
			if segmentData.EarliestTimestamp == 0 {
				segmentData.EarliestTimestamp = time
			}
			if segmentData.LatestTimestamp < time {
				segmentData.LatestTimestamp = time
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
