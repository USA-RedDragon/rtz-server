package logparser

import (
	"bufio"
	"compress/bzip2"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	v1dot4 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1.4"
	"github.com/USA-RedDragon/rtz-server/internal/storage"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/klauspost/compress/zstd"
	"github.com/mattn/go-nulltype"
	"github.com/puzpuzpuz/xsync/v3"
	"gorm.io/gorm"
)

const QueueDepth = 100

type LogQueue struct {
	config          *config.Config
	db              *gorm.DB
	storage         storage.Storage
	queue           chan work
	closeChan       chan any
	metrics         *metrics.Metrics
	activeJobsCount *xsync.Counter
	activeJobs      *xsync.MapOf[string, *work]
}

type work struct {
	path      string
	dongleID  string
	routeInfo v1dot4.RouteInfo
}

func NewLogQueue(config *config.Config, db *gorm.DB, storage storage.Storage, metrics *metrics.Metrics) *LogQueue {
	return &LogQueue{
		config:          config,
		db:              db,
		queue:           make(chan work, QueueDepth),
		closeChan:       make(chan any),
		metrics:         metrics,
		activeJobsCount: xsync.NewCounter(),
		activeJobs:      xsync.NewMapOf[string, *work](),
		storage:         storage,
	}
}

func (q *LogQueue) Start() {
	for work := range q.queue {
		_, ok := q.activeJobs.Load(work.dongleID)
		if ok {
			// If we already have a job for this dongle, we can't start another one
			q.queue <- work
			continue
		}

		if uint(q.activeJobsCount.Value()) < q.config.ParallelLogParsers {
			q.activeJobsCount.Inc()
			go func() {
				q.activeJobs.Store(work.dongleID, &work)
				err := q.processLog(q.db, q.storage, work)
				if err != nil {
					slog.Error("Error processing log", "log", work.path, "err", err)
				}
				q.activeJobs.Delete(work.dongleID)
				q.activeJobsCount.Dec()
			}()
		} else {
			q.queue <- work
		}
		if q.metrics != nil {
			q.metrics.SetLogParserActiveJobs(float64(q.activeJobsCount.Value()))
			q.metrics.SetLogParserQueueSize(float64(len(q.queue)))
		}
	}
	q.closeChan <- struct{}{}
}

func (q *LogQueue) Stop() {
	close(q.queue)
	<-q.closeChan
}

func (q *LogQueue) AddLog(path string, dongleID string, routeInfo v1dot4.RouteInfo) {
	q.queue <- work{path: path, dongleID: dongleID, routeInfo: routeInfo}
}

func (q *LogQueue) processLog(db *gorm.DB, storage storage.Storage, work work) error {
	rt, err := storage.Open(filepath.Join(work.dongleID, work.path))
	if err != nil {
		if q.metrics != nil {
			q.metrics.IncrementLogParserErrors(work.dongleID, "open_file")
		}
		slog.Error("Error opening file", "err", err)
		return err
	}
	defer rt.Close()

	device, err := models.FindDeviceByDongleID(q.db, work.dongleID)
	if err != nil {
		if q.metrics != nil {
			q.metrics.IncrementLogParserErrors(work.dongleID, "find_device")
		}
		slog.Error("Error finding device by dongle ID", "dongleID", work.dongleID, "err", err)
		return err
	}

	bufReader := bufio.NewReader(rt)
	var decompressedReader io.Reader

	switch filepath.Ext(work.path) {
	case ".zst":
		decompressedReader, err = zstd.NewReader(bufReader)
		if err != nil {
			if q.metrics != nil {
				q.metrics.IncrementLogParserErrors(work.dongleID, "new_zstd_reader")
			}
			slog.Error("Error creating new zstd reader", "err", err)
			return err
		}
	case ".bz2":
		decompressedReader = bzip2.NewReader(bufReader)
	default:
		if q.metrics != nil {
			q.metrics.IncrementLogParserErrors(work.dongleID, "unsupported_file_extension")
		}
		slog.Error("Unsupported file extension", "ext", filepath.Ext(work.path))
		return errors.New("unsupported file extension")
	}

	segmentData, err := DecodeSegmentData(decompressedReader)
	if err != nil {
		if q.metrics != nil {
			q.metrics.IncrementLogParserErrors(work.dongleID, "decode_segment_data")
		}
		slog.Error("Error decoding segment data", "err", err)
		return err
	}

	// We need to associate a segment with a route...
	route, err := models.FindRouteForSegment(db, device.ID, work.routeInfo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			route = models.Route{
				DeviceID:        device.ID,
				RouteID:         work.routeInfo.Route,
				GitBranch:       segmentData.GitBranch,
				GitRemote:       segmentData.GitRemote,
				GitDirty:        segmentData.GitDirty,
				GitCommit:       segmentData.GitCommit,
				InitLogMonoTime: segmentData.InitLogMonoTime,
				Platform:        segmentData.CarModel,
				Radar:           segmentData.Radar,
				Version:         segmentData.Version,
			}
		} else {
			if q.metrics != nil {
				q.metrics.IncrementLogParserErrors(work.dongleID, "find_route_for_segment")
			}
			slog.Error("Error finding route for segment", "err", err)
			return err
		}
	}

	if route.FirstClockLogMonoTime == 0 && segmentData.FirstClockLogMonoTime != 0 {
		route.FirstClockLogMonoTime = segmentData.FirstClockLogMonoTime
	}
	if route.FirstClockWallTimeNanos == 0 && segmentData.FirstClockWallTimeNanos != 0 {
		route.FirstClockWallTimeNanos = segmentData.FirstClockWallTimeNanos
	}
	numGPSLocs := len(segmentData.GPSLocations)
	if numGPSLocs > 0 {
		if route.StartTime.IsZero() {
			route.StartLat = segmentData.GPSLocations[0].Latitude
			route.StartLng = segmentData.GPSLocations[0].Longitude
			route.StartTime = time.Unix(0, int64(route.GetWallTimeFromBootTime(segmentData.GPSLocations[0].LogMonoTime)))
		}

		if !device.LastGPSTime.Valid() ||
			route.GetWallTimeFromBootTime(segmentData.GPSLocations[numGPSLocs-1].LogMonoTime) > uint64(device.LastGPSTime.TimeValue().UnixNano()) {
			latestTimeStamp := time.Unix(0, int64(route.GetWallTimeFromBootTime(segmentData.GPSLocations[numGPSLocs-1].LogMonoTime)))
			err := q.db.Model(&device).
				Updates(models.Device{
					LastGPSTime:    nulltype.NullTimeOf(latestTimeStamp),
					LastGPSLat:     nulltype.NullFloat64Of(segmentData.EndCoordinates.Latitude),
					LastGPSLng:     nulltype.NullFloat64Of(segmentData.EndCoordinates.Longitude),
					LastGPSBearing: nulltype.NullFloat64Of(segmentData.EndCoordinates.Bearing),
					LastGPSSpeed:   nulltype.NullFloat64Of(segmentData.EndCoordinates.SpeedMetersPerSecond),
				}).Error
			if err != nil {
				if q.metrics != nil {
					q.metrics.IncrementLogParserErrors(work.dongleID, "update_device")
				}
				slog.Error("Error updating device", "err", err)
				return err
			}
		}

		for i := 0; i < numGPSLocs; i++ {
			slog.Debug("Processing GPS location", "i", i)
			var lastGPS GpsCoordinates
			if i == 0 {
				if !segmentData.StartOfRoute {
					lastGPS = GpsCoordinates{
						Latitude:             device.LastGPSLat.Float64Value(),
						Longitude:            device.LastGPSLng.Float64Value(),
						SpeedMetersPerSecond: device.LastGPSSpeed.Float64Value(),
					}
				} else {
					// First entry in route, distance is zero
					continue
				}
			} else {
				lastGPS = segmentData.GPSLocations[i-1]
			}
			gps := segmentData.GPSLocations[i]
			slog.Info("Last GPS", "lat", lastGPS.Latitude, "lng", lastGPS.Longitude, "speed", lastGPS.SpeedMetersPerSecond)
			slog.Info("Current GPS", "lat", gps.Latitude, "lng", gps.Longitude, "speed", gps.SpeedMetersPerSecond)
			dist := utils.Haversine(lastGPS.Latitude, lastGPS.Longitude, gps.Latitude, gps.Longitude)
			slog.Info("Distance", "distance", dist)
			// How do we know if the distance is accurate and should be added?
			if dist > 2 && gps.SpeedMetersPerSecond > 1 {
				slog.Info("Distance is outside accuracy zone, adding to total length")
				segmentData.GPSLocations[i].Distance = dist
				route.Length += segmentData.GPSLocations[i].Distance
			} else {
				slog.Info("Distance is inside accuracy zone, not adding to total length", "distance", dist)
			}
			slog.Info("Total length", "length", route.Length)
		}
	}

	// TODO: Store gps data on route

	// route.SegmentStartTimes = append(route.SegmentStartTimes, route.GetWallTimeFromBootTime(segmentData.InitLogMonoTime))
	// route.SegmentEndTimes = append(route.SegmentEndTimes, route.GetWallTimeFromBootTime(segmentData.EndLogMonoTime))

	if segmentData.EndOfRoute {
		route.EndLat = segmentData.EndCoordinates.Latitude
		route.EndLng = segmentData.EndCoordinates.Longitude
		route.EndTime = time.Unix(0, int64(route.GetWallTimeFromBootTime(segmentData.EndLogMonoTime)))
		route.AllSegmentsProcessed = true
		// TODO: URL
	}

	return db.Save(&route).Error
}
