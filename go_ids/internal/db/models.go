package db

import (
	"time"

	"gorm.io/gorm"
)

// Alert represents a security alert detected by the IDS
type Alert struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"index" json:"timestamp"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	SourceIP   string  `gorm:"index" json:"source_ip"`
	DestIP     string  `json:"dest_ip"`
	Type       string  `json:"type"`       // e.g., "PortScan", "DDoS"
	Confidence float32 `json:"confidence"` // 0.0 - 1.0
	IsRead     bool    `gorm:"default:false" json:"is_read"`
	Payload    string  `gorm:"type:text" json:"payload"` // 新增：保存攻击报文/特征载荷
}
