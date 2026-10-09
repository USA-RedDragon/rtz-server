package logparser_test

import (
	"bytes"
	"math"
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/USA-RedDragon/rtz-server/internal/cereal"
	"github.com/USA-RedDragon/rtz-server/internal/logparser"
)

func setMeasurement(t *testing.T, m cereal.LiveLocationKalman_Measurement, values ...float64) {
	t.Helper()
	list, err := m.NewValue(int32(len(values)))
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range values {
		list.Set(i, v)
	}
	m.SetValid(true)
}

func kalmanEvent(t *testing.T, buf *bytes.Buffer, x, y, z, lat, lon float64) {
	t.Helper()
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	event, err := cereal.NewRootEvent(seg)
	if err != nil {
		t.Fatal(err)
	}
	event.SetLogMonoTime(1)
	kalman, err := event.NewLiveLocationKalmanDEPRECATED()
	if err != nil {
		t.Fatal(err)
	}
	ecef, err := kalman.NewPositionECEF()
	if err != nil {
		t.Fatal(err)
	}
	setMeasurement(t, ecef, x, y, z)
	geodetic, err := kalman.NewPositionGeodetic()
	if err != nil {
		t.Fatal(err)
	}
	setMeasurement(t, geodetic, lat*math.Pi/180, lon*math.Pi/180, 0)
	if err := capnp.NewEncoder(buf).Encode(msg); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeSegmentDataKalman(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	kalmanEvent(t, &buf, 0, 0, 0, 10, 20)
	kalmanEvent(t, &buf, 3, 4, 0, 11, 21)

	data, err := logparser.DecodeSegmentData(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.KalmanPositions) != 2 {
		t.Fatalf("got %d kalman positions, want 2", len(data.KalmanPositions))
	}
	if data.TotalDistance != 5 {
		t.Errorf("got total distance %v, want 5", data.TotalDistance)
	}
	last := data.KalmanPositions[1]
	if math.Abs(last.Latitude-11) > 1e-9 || math.Abs(last.Longitude-21) > 1e-9 {
		t.Errorf("got lat/lon %v/%v, want 11/21", last.Latitude, last.Longitude)
	}
}
