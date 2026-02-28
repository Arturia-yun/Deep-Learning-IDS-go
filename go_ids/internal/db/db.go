package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the SQLite database connection
func InitDB(dbPath string) error {
	var err error

	// Ensure directory exists (basic check, though sqlite usually creates file)
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for db: %w", err)
	}

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // Reduce log noise
	}

	DB, err = gorm.Open(sqlite.Open(absPath), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Connection Pool Settings
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto Migrate
	err = DB.AutoMigrate(&Alert{})
	if err != nil {
		return fmt.Errorf("failed to migrate database schema: %w", err)
	}

	// 注入测试数据或清理错误数据
	var badCount int64
	DB.Model(&Alert{}).Where("confidence > 1.0").Count(&badCount)
	if badCount > 0 {
		// 如果发现此前的脏数据，直接清空整个表并重新注入
		DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Alert{})
		seedMockAlerts()
	} else {
		var totalCount int64
		DB.Model(&Alert{}).Count(&totalCount)
		if totalCount == 0 {
			seedMockAlerts()
		}
	}

	return nil
}

// seedMockAlerts 专门为用户展示提供极具实战细节的演习用攻击日志
func seedMockAlerts() {
	now := time.Now()
	alerts := []Alert{
		{CreatedAt: now.Add(-12 * time.Hour), SourceIP: "104.26.6.57", DestIP: "192.168.1.100", Type: "DDoS", Confidence: 0.99, Payload: "GET / HTTP/1.1\r\nHost: target.com\r\nUser-Agent: Hulk/1.0\r\n\r\n"},
		{CreatedAt: now.Add(-11 * time.Hour), SourceIP: "45.33.18.12", DestIP: "192.168.1.50", Type: "Web Attack", Confidence: 0.94, Payload: "POST /login.php HTTP/1.1\r\nContent-Type: application/x-www-form-urlencoded\r\n\r\nusername=admin' OR '1'='1&password=123"},
		{CreatedAt: now.Add(-9 * time.Hour), SourceIP: "185.199.108.133", DestIP: "192.168.1.10", Type: "PortScan", Confidence: 0.88, Payload: "SYN Stealth Scan (Nmap) Packet Signature Detected. Window Size = 1024."},
		{CreatedAt: now.Add(-8 * time.Hour), SourceIP: "117.72.62.10", DestIP: "192.168.1.100", Type: "Brute Force", Confidence: 0.92, Payload: "SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1\npassword matching failed"},
		{CreatedAt: now.Add(-7 * time.Hour), SourceIP: "52.220.222.172", DestIP: "192.168.1.15", Type: "Bot", Confidence: 0.85, Payload: "GET /c2_command.php?id=9928 HTTP/1.1\r\nUser-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)"},
		{CreatedAt: now.Add(-6 * time.Hour), SourceIP: "151.101.193.91", DestIP: "192.168.1.50", Type: "Web Attack", Confidence: 0.96, Payload: "GET /../../../../etc/passwd HTTP/1.1\r\nHost: example.com\r\n"},
		{CreatedAt: now.Add(-5 * time.Hour), SourceIP: "142.250.71.131", DestIP: "192.168.1.20", Type: "PortScan", Confidence: 0.78, Payload: "UDP Scan detected on ports: 53, 161, 123"},
		{CreatedAt: now.Add(-4 * time.Hour), SourceIP: "3.168.86.75", DestIP: "192.168.1.100", Type: "DDoS", Confidence: 0.97, Payload: "Volumetric SYN Flood. Packet Rate: 45000 pps"},
		{CreatedAt: now.Add(-3 * time.Hour), SourceIP: "180.105.204.112", DestIP: "192.168.1.12", Type: "Brute Force", Confidence: 0.89, Payload: "FTP Login failed: 530 Login incorrect. User: anonymous"},
		{CreatedAt: now.Add(-2 * time.Hour), SourceIP: "58.216.102.31", DestIP: "192.168.1.50", Type: "Web Attack", Confidence: 0.91, Payload: "GET /index.php?id=1 UNION SELECT null, version() HTTP/1.1\r\nHost: target\r\n"},
		{CreatedAt: now.Add(-1 * time.Hour), SourceIP: "222.186.176.192", DestIP: "192.168.1.100", Type: "Bot", Confidence: 0.82, Payload: "Mirai Botnet signature detected. Hardcoded credential guess."},
		{CreatedAt: now.Add(-20 * time.Minute), SourceIP: "114.237.67.68", DestIP: "192.168.1.50", Type: "Web Attack", Confidence: 0.95, Payload: "GET /?q=<script>alert('XSS')</script> HTTP/1.1\r\nHost: target\r\n"},
	}

	for _, a := range alerts {
		DB.Create(&a)
	}
}

// CreateAlert saves a new alert to the database
func CreateAlert(alert *Alert) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	result := DB.Create(alert)
	return result.Error
}

// GetRecentAlerts retrieves the latest N alerts
func GetRecentAlerts(limit int) ([]Alert, error) {
	var alerts []Alert
	result := DB.Order("created_at desc").Limit(limit).Find(&alerts)
	return alerts, result.Error
}

// StatsPoint represents a single data point in the chart
type StatsPoint struct {
	Label string `json:"label"` // Time label (e.g., "10:00", "Mon", "05-12")
	Count int    `json:"count"`
}

// GetThreatStats aggregates threat counts based on the time range
func GetThreatStats(rangeType string) ([]StatsPoint, error) {
	var points []StatsPoint
	var err error
	var rows *sql.Rows
	now := time.Now()

	/*
	   Strategy:
	   1. Generate complete time buckets in Go to ensure zero-filling.
	   2. Query DB for counts grouped by formatted time string.
	   3. Map DB results to the buckets.
	*/

	if rangeType == "Day" {
		// Past 24 Hours
		// Map: "15:00" -> Count
		counts := make(map[string]int)

		// SQLite: strftime('%H:00', created_at, 'localtime') might be needed depending on timezone
		// For simplicity/portability, we trust system time.
		// Select strftime('%H:00', created_at) as label, count(*) as count
		// from alerts where created_at > ? group by label

		rows, err = DB.Model(&Alert{}).
			Select("strftime('%H:00', created_at, 'localtime') as label, count(*) as count").
			Where("created_at > ?", now.Add(-24*time.Hour)).
			Group("label").
			Rows()

		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var label string
			var count int
			rows.Scan(&label, &count)
			counts[label] = count
		}

		// Generate last 24 buckets properly
		for i := 23; i >= 0; i-- {
			t := now.Add(time.Duration(-i) * time.Hour)
			label := t.Format("15:00") // HH:00
			points = append(points, StatsPoint{Label: label, Count: counts[label]})
		}

	} else if rangeType == "Week" {
		// Past 7 Days
		counts := make(map[string]int)

		rows, err = DB.Model(&Alert{}).
			Select("strftime('%Y-%m-%d', created_at, 'localtime') as label, count(*) as count").
			Where("created_at > ?", now.Add(-7*24*time.Hour)).
			Group("label").
			Rows()

		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var label string
			var count int
			rows.Scan(&label, &count)
			counts[label] = count
		}

		// Generate last 7 days buckets
		for i := 6; i >= 0; i-- {
			t := now.AddDate(0, 0, -i)
			key := t.Format("2006-01-02")
			displayLabel := t.Format("Mon") // Mon, Tue...
			points = append(points, StatsPoint{Label: displayLabel, Count: counts[key]})
		}

	} else { // Month
		// Past 30 Days
		counts := make(map[string]int)

		rows, err = DB.Model(&Alert{}).
			Select("strftime('%Y-%m-%d', created_at, 'localtime') as label, count(*) as count").
			Where("created_at > ?", now.Add(-30*24*time.Hour)).
			Group("label").
			Rows()

		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var label string
			var count int
			rows.Scan(&label, &count)
			counts[label] = count
		}

		for i := 29; i >= 0; i-- {
			t := now.AddDate(0, 0, -i)
			key := t.Format("2006-01-02")
			displayLabel := t.Format("02") // Day of month
			points = append(points, StatsPoint{Label: displayLabel, Count: counts[key]})
		}
	}

	return points, nil
}
