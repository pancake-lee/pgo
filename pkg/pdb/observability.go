package pdb

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

const queryStartKey = "pgo_query_observation_start"

var queryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "pgo_db_query_duration_seconds",
	Help:    "Database operation latency observed by GORM callbacks.",
	Buckets: prometheus.DefBuckets,
}, []string{"operation", "result"})

func init() {
	prometheus.MustRegister(
		queryDuration,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "pgo_db_connections_open",
			Help: "Number of established database connections.",
		}, func() float64 { return float64(getDBStats().OpenConnections) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "pgo_db_connections_in_use",
			Help: "Number of database connections currently in use.",
		}, func() float64 { return float64(getDBStats().InUse) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "pgo_db_connections_idle",
			Help: "Number of idle database connections.",
		}, func() float64 { return float64(getDBStats().Idle) }),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "pgo_db_connection_wait_total",
			Help: "Total number of waits for a database connection.",
		}, func() float64 { return float64(getDBStats().WaitCount) }),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "pgo_db_connection_wait_duration_seconds_total",
			Help: "Total time blocked waiting for a database connection.",
		}, func() float64 { return getDBStats().WaitDuration.Seconds() }),
	)
}

func registerQueryObservability(db *gorm.DB) error {
	if err := db.Callback().Create().Before("*").Register("pgo:observe_create_before", observeQueryStart); err != nil {
		return err
	}
	if err := db.Callback().Create().After("*").Register("pgo:observe_create_after", observeQueryEnd("create")); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("*").Register("pgo:observe_delete_before", observeQueryStart); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("*").Register("pgo:observe_delete_after", observeQueryEnd("delete")); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("*").Register("pgo:observe_query_before", observeQueryStart); err != nil {
		return err
	}
	if err := db.Callback().Query().After("*").Register("pgo:observe_query_after", observeQueryEnd("query")); err != nil {
		return err
	}
	if err := db.Callback().Raw().Before("*").Register("pgo:observe_raw_before", observeQueryStart); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("*").Register("pgo:observe_raw_after", observeQueryEnd("raw")); err != nil {
		return err
	}
	if err := db.Callback().Row().Before("*").Register("pgo:observe_row_before", observeQueryStart); err != nil {
		return err
	}
	if err := db.Callback().Row().After("*").Register("pgo:observe_row_after", observeQueryEnd("row")); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("*").Register("pgo:observe_update_before", observeQueryStart); err != nil {
		return err
	}
	return db.Callback().Update().After("*").Register("pgo:observe_update_after", observeQueryEnd("update"))
}

func observeQueryStart(db *gorm.DB) {
	db.InstanceSet(queryStartKey, time.Now())
}

func observeQueryEnd(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startedAt, ok := db.InstanceGet(queryStartKey)
		if !ok {
			return
		}
		startTime, ok := startedAt.(time.Time)
		if !ok {
			return
		}
		result := "ok"
		if db.Error != nil {
			result = "error"
		}
		queryDuration.WithLabelValues(operation, result).Observe(time.Since(startTime).Seconds())
	}
}

func getDBStats() (stats databaseStats) {
	db, err := GetDB()
	if err != nil {
		return databaseStats{}
	}
	current := db.Stats()
	return databaseStats{
		OpenConnections: current.OpenConnections,
		InUse:           current.InUse,
		Idle:            current.Idle,
		WaitCount:       current.WaitCount,
		WaitDuration:    current.WaitDuration,
	}
}

type databaseStats struct {
	OpenConnections int
	InUse           int
	Idle            int
	WaitCount       int64
	WaitDuration    time.Duration
}
