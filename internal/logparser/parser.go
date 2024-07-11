package logparser

import (
	"errors"
	"io"

	"capnproto.org/go/capnp/v3"
	"github.com/USA-RedDragon/rtz-server/internal/cereal"
)

type GpsCoordinates struct {
	Latitude  float64
	Longitude float64
}

type SegmentData struct {
	GPSLocations      []GpsCoordinates
	EarliestTimestamp uint64
	LatestTimestamp   uint64
	CarModel          string
	GitRemote         string
	GitBranch         string
}

func DecodeSegmentData(reader io.Reader) (SegmentData, error) {
	var segmentData SegmentData

	decoder := capnp.NewDecoder(reader)
	for {
		msg, err := decoder.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return SegmentData{}, err
			}
			break
		}
		event, err := cereal.ReadRootEvent(msg)
		if err != nil {
			return SegmentData{}, err
		}
		// We're definitely not going to be handling every event type, so we can ignore the exhaustive linter warning
		//nolint:golint,exhaustive
		switch event.Which() {
		case cereal.Event_Which_liveLocationKalman:
			liveLocationKalman, err := event.LiveLocationKalman()
			if err != nil {
				return SegmentData{}, err
			}
			position, err := liveLocationKalman.PositionGeodetic()
			if err != nil {
				return SegmentData{}, err
			}
			if !liveLocationKalman.GpsOK() {
				continue
			}
			values, err := position.Value()
			if err != nil {
				return SegmentData{}, err
			}
			segmentData.GPSLocations = append(segmentData.GPSLocations, GpsCoordinates{
				Latitude:  values.At(0),
				Longitude: values.At(1),
			})
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
			if segmentData.CarModel != "" && segmentData.GitRemote != "" && segmentData.GitBranch != "" {
				continue
			}
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
