package domain

import "time"

// OutageRecord 停电统计记录：由故障事件复电时生成。
type OutageRecord struct {
	ID              string    `json:"id"`
	FaultEventID    string    `json:"faultEventId"`
	FeederID        string    `json:"feederId"`
	FeederName      string    `json:"feederName"`
	OutageStart     time.Time `json:"outageStart"`
	OutageEnd       time.Time `json:"outageEnd"`
	DurationMinutes int       `json:"durationMinutes"`
	// LongOutage 是否超过长时停电阈值（默认 120 分钟）。
	LongOutage bool      `json:"longOutage"`
	CreatedAt  time.Time `json:"createdAt"`
}

// LongOutageThresholdMinutes 长时停电阈值（分钟），与 config 保持一致。
const LongOutageThresholdMinutes = 120

// NewOutageRecord 根据已复电的故障事件构造停电记录。
// 停电开始时间为定位时间，结束时间为复电时间，时长=复电-定位。
func NewOutageRecord(id string, event *FaultEvent, now time.Time) *OutageRecord {
	start := event.LocatedAt
	end := event.RestoredAt
	if end.IsZero() {
		end = now
	}
	duration := end.Sub(start)
	if duration < 0 {
		duration = 0
	}
	minutes := int(duration.Minutes())
	return &OutageRecord{
		ID:              id,
		FaultEventID:    event.ID,
		FeederID:        event.FeederID,
		FeederName:      event.FeederName,
		OutageStart:     start,
		OutageEnd:       end,
		DurationMinutes: minutes,
		LongOutage:      minutes >= LongOutageThresholdMinutes,
		CreatedAt:       now,
	}
}

// OutageSummary 停电统计汇总。
type OutageSummary struct {
	TotalRecords    int                `json:"totalRecords"`
	TotalMinutes    int                `json:"totalMinutes"`
	AvgMinutes      float64            `json:"avgMinutes"`
	MaxMinutes      int                `json:"maxMinutes"`
	LongOutageCount int                `json:"longOutageCount"`
	ByFeeder        []FeederOutageStat `json:"byFeeder"`
}

// FeederOutageStat 单条线路的停电统计。
type FeederOutageStat struct {
	FeederID     string `json:"feederId"`
	FeederName   string `json:"feederName"`
	RecordCount  int    `json:"recordCount"`
	TotalMinutes int    `json:"totalMinutes"`
	LongOutages  int    `json:"longOutages"`
}
