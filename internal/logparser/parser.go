package logparser

import (
	"errors"
	"fmt"
	"io"
	"math"

	"capnproto.org/go/capnp/v3"
	"github.com/USA-RedDragon/rtz-server/internal/cereal"
)

// calculateECEFDistance calculates the Euclidean distance between two ECEF positions in meters
func calculateECEFDistance(pos1, pos2 KalmanPosition) float64 {
	dx := pos2.X - pos1.X
	dy := pos2.Y - pos1.Y
	dz := pos2.Z - pos1.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

type GpsCoordinates struct {
	Latitude  float64
	Longitude float64
}

type KalmanPosition struct {
	X         float64 // ECEF X coordinate in meters
	Y         float64 // ECEF Y coordinate in meters
	Z         float64 // ECEF Z coordinate in meters
	Latitude  float64 // Geodetic latitude in degrees
	Longitude float64 // Geodetic longitude in degrees
	Valid     bool
}

type SegmentData struct {
	EndLogMonoTime          uint64
	FirstClockWallTimeNanos uint64
	FirstClockLogMonoTime   uint64
	GitDirty                bool
	GitCommit               string
	Version                 string
	DongleID                string
	InitLogMonoTime         uint64
	DeviceType              cereal.InitData_DeviceType
	CarModel                string
	GitRemote               string
	GitBranch               string
	StartOfRoute            bool
	EndOfRoute              bool
	KalmanPositions         []KalmanPosition
	TotalDistance           float64
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
		case cereal.Event_Which_liveLocationKalman:
			liveLocation, err := event.LiveLocationKalman()
			if err != nil {
				return SegmentData{}, err
			}

			positionECEF, err := liveLocation.PositionECEF()
			if err == nil && positionECEF.Valid() {
				values, err := positionECEF.Value()
				if err == nil && values.Len() >= 3 {
					positionGeodetic, err := liveLocation.PositionGeodetic()
					var lat, lon float64
					if err == nil && positionGeodetic.Valid() {
						geoValues, err := positionGeodetic.Value()
						if err == nil && geoValues.Len() >= 2 {
							lat = geoValues.At(0) * 180.0 / math.Pi // Convert radians to degrees
							lon = geoValues.At(1) * 180.0 / math.Pi // Convert radians to degrees
						}
					}

					position := KalmanPosition{
						X:         values.At(0),
						Y:         values.At(1),
						Z:         values.At(2),
						Latitude:  lat,
						Longitude: lon,
						Valid:     true,
					}

					if len(segmentData.KalmanPositions) > 0 {
						lastPos := segmentData.KalmanPositions[len(segmentData.KalmanPositions)-1]
						distance := calculateECEFDistance(lastPos, position)
						segmentData.TotalDistance += distance
					}
					segmentData.KalmanPositions = append(segmentData.KalmanPositions, position)
				}
			}
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
