package pdb

import (
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestDatabaseMetrics(t *testing.T) {
	if err := InitSqlite(filepath.Join(t.TempDir(), "metrics.db")); err != nil {
		t.Fatal(err)
	}
	type record struct {
		ID   int `gorm:"primaryKey"`
		Name string
	}
	if err := gDB.AutoMigrate(&record{}); err != nil {
		t.Fatal(err)
	}
	if err := gDB.Create(&record{Name: "one"}).Error; err != nil {
		t.Fatal(err)
	}
	var recordList []record
	if err := gDB.Find(&recordList).Error; err != nil {
		t.Fatal(err)
	}

	metricFamilyList, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	requiredMetricMap := map[string]bool{
		"pgo_db_query_duration_seconds":                 false,
		"pgo_db_connections_open":                       false,
		"pgo_db_connections_in_use":                     false,
		"pgo_db_connections_idle":                       false,
		"pgo_db_connection_wait_total":                  false,
		"pgo_db_connection_wait_duration_seconds_total": false,
	}
	for _, metricFamily := range metricFamilyList {
		if _, ok := requiredMetricMap[metricFamily.GetName()]; ok {
			requiredMetricMap[metricFamily.GetName()] = true
		}
	}
	for metricName, found := range requiredMetricMap {
		if !found {
			t.Errorf("metric %s was not gathered", metricName)
		}
	}
}
