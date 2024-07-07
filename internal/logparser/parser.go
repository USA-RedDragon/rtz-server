package logparser

import (
	"io"

	"capnproto.org/go/capnp/v3"
	"github.com/USA-RedDragon/connect-server/internal/cereal"
)

type GpsCoordinates struct {
	Latitude  float64
	Longitude float64
}

type SegmentData struct {
	GPSLocations      []GpsCoordinates
	EarliestTimestamp uint64
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
			if err != io.EOF {
				return SegmentData{}, err
			} else {
				break
			}
		}
		event, err := cereal.ReadRootEvent(msg)
		if err != nil {
			return SegmentData{}, err
		}

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
			if segmentData.EarliestTimestamp != 0 {
				continue
			}
			clocks, err := event.Clocks()
			if err != nil {
				return SegmentData{}, err
			}
			segmentData.EarliestTimestamp = clocks.WallTimeNanos()
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
				switch key {
				case "CarModel":
					segmentData.CarModel = string(val)
				}
			}
		}
	}

	return segmentData, nil
}
