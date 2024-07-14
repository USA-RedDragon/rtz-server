package logparser

import (
	"bufio"
	"compress/bzip2"
	"log/slog"
	"os"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/mattn/go-nulltype"
	"github.com/puzpuzpuz/xsync/v3"
	"gorm.io/gorm"
)

const QueueDepth = 100

type LogQueue struct {
	config     *config.Config
	db         *gorm.DB
	queue      chan work
	closeChan  chan any
	metrics    *metrics.Metrics
	activeJobs *xsync.Counter
}

type work struct {
	path     string
	dongleID string
}

func NewLogQueue(config *config.Config, db *gorm.DB, metrics *metrics.Metrics) *LogQueue {
	return &LogQueue{
		config:     config,
		db:         db,
		queue:      make(chan work, QueueDepth),
		closeChan:  make(chan any),
		metrics:    metrics,
		activeJobs: xsync.NewCounter(),
	}
}

func (q *LogQueue) Start() {
	for work := range q.queue {
		if uint(q.activeJobs.Value()) < q.config.ParallelLogParsers {
			q.activeJobs.Inc()
			go func() {
				err := q.processLog(work)
				if err != nil {
					slog.Error("Error processing log", "log", work.path, "err", err)
				}
				q.activeJobs.Dec()
			}()
		} else {
			q.queue <- work
		}
		q.metrics.SetLogParserActiveJobs(float64(q.activeJobs.Value()))
		q.metrics.SetLogParserQueueSize(float64(len(q.queue)))
	}
	q.closeChan <- struct{}{}
}

func (q *LogQueue) Stop() {
	close(q.queue)
	<-q.closeChan
}

func (q *LogQueue) AddLog(path string, dongleID string) {
	q.queue <- work{path: path, dongleID: dongleID}
}

func (q *LogQueue) processLog(work work) error {
	rt, err := os.Open(work.path)
	if err != nil {
		q.metrics.IncrementLogParserErrors(work.dongleID, "open_file")
		slog.Error("Error opening file", "err", err)
		return err
	}
	defer rt.Close()

	device, err := models.FindDeviceByDongleID(q.db, work.dongleID)
	if err != nil {
		q.metrics.IncrementLogParserErrors(work.dongleID, "find_device")
		slog.Error("Error finding device by dongle ID", "dongleID", work.dongleID, "err", err)
		return err
	}

	segmentData, err := DecodeSegmentData(bzip2.NewReader(bufio.NewReader(rt)))
	if err != nil {
		q.metrics.IncrementLogParserErrors(work.dongleID, "decode_segment_data")
		slog.Error("Error decoding segment data", "err", err)
		return err
	}
	slog.Info("Segment data", "numGpsPoints", len(segmentData.GPSLocations), "earliestTimestamp", segmentData.EarliestTimestamp, "latestTimestamp", segmentData.LatestTimestamp, "carModel", segmentData.CarModel, "gitRemote", segmentData.GitRemote, "gitBranch", segmentData.GitBranch)
	if (!device.LastGPSTime.Valid() || segmentData.LatestTimestamp > uint64(device.LastGPSTime.TimeValue().UnixNano())) && len(segmentData.GPSLocations) > 0 {
		latestTimeStamp := time.Unix(0, int64(segmentData.LatestTimestamp))
		err := q.db.Model(&device).
			Updates(models.Device{
				LastGPSTime:     nulltype.NullTimeOf(latestTimeStamp),
				LastGPSLat:      nulltype.NullFloat64Of(segmentData.GPSLocations[len(segmentData.GPSLocations)-1].Latitude),
				LastGPSLng:      nulltype.NullFloat64Of(segmentData.GPSLocations[len(segmentData.GPSLocations)-1].Longitude),
				LastGPSBearing:  nulltype.NullFloat64Of(segmentData.GPSLocations[len(segmentData.GPSLocations)-1].Bearing),
				LastGPSSpeed:    nulltype.NullFloat64Of(segmentData.GPSLocations[len(segmentData.GPSLocations)-1].SpeedMetersPerSecond),
				LastGPSAccuracy: nulltype.NullFloat64Of(segmentData.GPSLocations[len(segmentData.GPSLocations)-1].AccuracyMeters),
			}).Error
		if err != nil {
			q.metrics.IncrementLogParserErrors(work.dongleID, "update_device")
			slog.Error("Error updating device", "err", err)
			return err
		}
	}

	return nil
}
