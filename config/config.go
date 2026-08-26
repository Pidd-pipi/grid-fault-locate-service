// Package config 提供服务的集中配置：端口、数据文件路径、日志级别、
// 电压等级、长时停电阈值与扫描周期等全局常量，并支持环境变量覆盖。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// 电压等级常量（前后端保持一致，前端定义于 web/enums.js）。
const (
	Voltage10kV = "10kV"
	Voltage20kV = "20kV"
	Voltage35kV = "35kV"
)

// VoltageLevels 是系统支持的电压等级全集，新增线路时校验用。
var VoltageLevels = []string{Voltage10kV, Voltage20kV, Voltage35kV}

// IsValidVoltageLevel 校验电压等级是否在支持集合内。
func IsValidVoltageLevel(v string) bool {
	for _, level := range VoltageLevels {
		if level == v {
			return true
		}
	}
	return false
}

const (
	// DefaultPort 服务默认监听端口。
	DefaultPort = "8080"
	// DefaultDataFile 默认 JSON 持久化文件路径。
	DefaultDataFile = "data/grid-fault-locate-data.json"
	// DefaultLongOutageMinutes 长时停电阈值（分钟），超过 2 小时进入关注清单。
	DefaultLongOutageMinutes = 120
	// DefaultSweepInterval 长时停电扫描定时任务的周期。
	DefaultSweepInterval = 10 * time.Minute
	// DefaultMaxAuditEntries 审计日志最多保留条数（防止无限增长）。
	DefaultMaxAuditEntries = 2000
	// DefaultRequestBodyLimit 请求体最大字节数。
	DefaultRequestBodyLimit = 1 << 20
	// DefaultLogLevel 默认结构化日志级别。
	DefaultLogLevel = "info"
)

// Config 汇总服务运行所需的全部配置。
type Config struct {
	// Port 监听端口，可由 PORT 环境变量覆盖。
	Port string
	// DataFile JSON 持久化文件路径，可由 DATA_FILE 环境变量覆盖。
	DataFile string
	// Persist 是否启用 JSON 文件持久化。
	Persist bool
	// LongOutageMinutes 长时停电判定阈值（分钟）。
	LongOutageMinutes int
	// SweepInterval 长时停电扫描周期。
	SweepInterval time.Duration
	// RequestBodyLimit 请求体大小上限。
	RequestBodyLimit int64
	// LogLevel 结构化日志级别：debug / info / warn / error。
	LogLevel string
}

// Default 返回一份默认配置。
func Default() Config {
	return Config{
		Port:              DefaultPort,
		DataFile:          DefaultDataFile,
		Persist:           true,
		LongOutageMinutes: DefaultLongOutageMinutes,
		SweepInterval:     DefaultSweepInterval,
		RequestBodyLimit:  DefaultRequestBodyLimit,
		LogLevel:          DefaultLogLevel,
	}
}

// Load 从环境变量读取配置并校验。
// 支持：PORT、DATA_FILE、PERSIST、LONG_OUTAGE_MINUTES、SWEEP_INTERVAL、
// REQUEST_BODY_LIMIT、LOG_LEVEL。
func Load() (Config, error) {
	cfg := Default()

	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("DATA_FILE"); v != "" {
		cfg.DataFile = v
	}
	if v := os.Getenv("PERSIST"); v != "" {
		b, err := parseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("PERSIST: %w", err)
		}
		cfg.Persist = b
		if !b {
			cfg.DataFile = ""
		}
	}
	if v := os.Getenv("LONG_OUTAGE_MINUTES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("LONG_OUTAGE_MINUTES must be a positive integer, got %q", v)
		}
		cfg.LongOutageMinutes = n
	}
	if v := os.Getenv("SWEEP_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("SWEEP_INTERVAL must be a positive duration (e.g. 10m), got %q", v)
		}
		cfg.SweepInterval = d
	}
	if v := os.Getenv("REQUEST_BODY_LIMIT"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("REQUEST_BODY_LIMIT must be a positive integer, got %q", v)
		}
		cfg.RequestBodyLimit = n
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(strings.TrimSpace(v))
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate 校验配置合法性，非法时返回错误。
func (c Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT must not be empty")
	}
	p, err := strconv.Atoi(c.Port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("PORT must be a number in [1,65535], got %q", c.Port)
	}
	if c.Persist && c.DataFile == "" {
		return fmt.Errorf("DATA_FILE must not be empty when persistence is enabled")
	}
	if c.LongOutageMinutes <= 0 {
		return fmt.Errorf("LongOutageMinutes must be positive, got %d", c.LongOutageMinutes)
	}
	if c.SweepInterval <= 0 {
		return fmt.Errorf("SweepInterval must be positive, got %s", c.SweepInterval)
	}
	if c.RequestBodyLimit <= 0 {
		return fmt.Errorf("RequestBodyLimit must be positive, got %d", c.RequestBodyLimit)
	}
	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL must be one of debug/info/warn/error, got %q", c.LogLevel)
	}
	return nil
}

// LongOutageThreshold 返回长时停电阈值的 time.Duration 表示。
func (c Config) LongOutageThreshold() time.Duration {
	return time.Duration(c.LongOutageMinutes) * time.Minute
}

func parseBool(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", v)
	}
}
