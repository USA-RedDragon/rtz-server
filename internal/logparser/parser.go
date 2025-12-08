package logparser

import (
	"errors"
	"fmt"
	"io"

	"capnproto.org/go/capnp/v3"
	"github.com/USA-RedDragon/rtz-server/internal/cereal"
)

type GpsCoordinates struct {
	Latitude             float64
	Longitude            float64
	LogMonoTime          uint64
	SpeedMetersPerSecond float64
	Bearing              float64
}

type AccelerometerData struct {
	X           float64
	Y           float64
	Z           float64
	LogMonoTime uint64
}

type GyroscopeData struct {
	X           float64
	Y           float64
	Z           float64
	LogMonoTime uint64
}

type SegmentData struct {
	GPSLocations            []GpsCoordinates
	AccelData               []AccelerometerData
	GyroData                []GyroscopeData
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

func DecodeSegmentData(reader io.Reader) (SegmentData, error) {
	var segmentData SegmentData

	decoder := capnp.NewDecoder(reader)
	for {
		msg, err := decoder.Decode()
		if err != nil {
			if !errors.Is(err, io.EOF) {
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
		case cereal.Event_Which_accelerometer, cereal.Event_Which_accelerometer2:
			var sensorData cereal.SensorEventData
			var err error
			switch event.Which() {
			case cereal.Event_Which_accelerometer:
				sensorData, err = event.Accelerometer()
			case cereal.Event_Which_accelerometer2:
				sensorData, err = event.Accelerometer2()
			}
			if err != nil {
				return SegmentData{}, err
			}
			accel, err := sensorData.Acceleration()
			if err != nil {
				return SegmentData{}, err
			}
			v, err := accel.V()
			if err != nil {
				return SegmentData{}, err
			}
			if v.Len() == 3 {
				x := float64(v.At(0))
				y := float64(v.At(1))
				z := float64(v.At(2))
				segmentData.AccelData = append(segmentData.AccelData, AccelerometerData{
					X:           x,
					Y:           y,
					Z:           z,
					LogMonoTime: event.LogMonoTime(),
				})
			}
		case cereal.Event_Which_gyroscope, cereal.Event_Which_gyroscope2:
			var sensorData cereal.SensorEventData
			var err error
			switch event.Which() {
			case cereal.Event_Which_gyroscope:
				sensorData, err = event.Gyroscope()
			case cereal.Event_Which_gyroscope2:
				sensorData, err = event.Gyroscope2()
			}
			if err != nil {
				return SegmentData{}, err
			}
			gyro, err := sensorData.Gyro()
			if err != nil {
				return SegmentData{}, err
			}
			v, err := gyro.V()
			if err != nil {
				return SegmentData{}, err
			}
			if v.Len() == 3 {
				x := float64(v.At(0))
				y := float64(v.At(1))
				z := float64(v.At(2))
				segmentData.GyroData = append(segmentData.GyroData, GyroscopeData{
					X:           x,
					Y:           y,
					Z:           z,
					LogMonoTime: event.LogMonoTime(),
				})
			}
		case cereal.Event_Which_gpsLocation, cereal.Event_Which_gpsLocationExternal:
			var gpsLocation cereal.GpsLocationData
			var err error
			switch event.Which() {
			case cereal.Event_Which_gpsLocation:
				gpsLocation, err = event.GpsLocation()
			case cereal.Event_Which_gpsLocationExternal:
				gpsLocation, err = event.GpsLocationExternal()
			}
			if err != nil {
				return SegmentData{}, err
			}
			gps := GpsCoordinates{
				Latitude:             gpsLocation.Latitude(),
				Longitude:            gpsLocation.Longitude(),
				SpeedMetersPerSecond: float64(gpsLocation.Speed()),
				Bearing:              float64(gpsLocation.BearingDeg()),
				LogMonoTime:          event.LogMonoTime(),
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
