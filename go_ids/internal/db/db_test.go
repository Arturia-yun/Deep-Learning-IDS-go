package db_test

import (
	"path/filepath"
	"testing"
	"time"

	"go-ids/internal/db"
)

func TestDB(t *testing.T) {
	// Setup temporary DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Test Create
	alert := &db.Alert{
		CreatedAt:  time.Now(),
		SourceIP:   "192.168.1.100",
		DestIP:     "10.0.0.1",
		Type:       "PortScan",
		Confidence: 0.95,
	}

	err = db.CreateAlert(alert)
	if err != nil {
		t.Errorf("Failed to create alert: %v", err)
	}

	// Test Retrieval
	alerts, err := db.GetRecentAlerts(10)
	if err != nil {
		t.Errorf("Failed to get alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].Type != "PortScan" {
		t.Errorf("Expected Type 'PortScan', got '%s'", alerts[0].Type)
	}

	// Cleanup happens automatically for t.TempDir
}
