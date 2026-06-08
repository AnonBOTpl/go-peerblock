package config

import (
	"time"

	"go-peerblock/updater"
)

// Config holds all application configuration.
type Config struct {
	ProtectionEnabled    bool             `json:"protection_enabled"`
	StartMinimized       bool             `json:"start_minimized"`
	StartWithSystem      bool             `json:"start_with_system"`
	NotificationsEnabled  bool             `json:"notifications_enabled"`
	MinimizeToTrayOnClose bool            `json:"minimize_to_tray_on_close"`
	WorkerCount         int              `json:"worker_count"`
	CacheSize           int              `json:"cache_size"`
	CacheTTL            time.Duration    `json:"cache_ttl"`
	UpdateInterval      time.Duration    `json:"update_interval"`
	LogLevel            string           `json:"log_level"`
	LogMaxSizeMB        int              `json:"log_max_size_mb"`
	Sources             []updater.Source `json:"sources"`
	Allowlist           []string         `json:"allowlist"`
	CustomRules         []string         `json:"custom_rules"`
	Language            string           `json:"language"`
}

// Defaults returns the default configuration.
func Defaults() *Config {
	return &Config{
		ProtectionEnabled:     true,
		StartMinimized:        false,
		StartWithSystem:       false,
		NotificationsEnabled:  true,
		MinimizeToTrayOnClose: false,
		WorkerCount:          0, // 0 = auto (NumCPU)
		CacheSize:            65536,
		CacheTTL:             5 * time.Minute,
		UpdateInterval:       24 * time.Hour,
		LogLevel:             "info",
		LogMaxSizeMB:         10,
		Sources:              updater.DefaultSources,
		Language:            "en",
		CustomRules: nil,
		Allowlist: []string{
			"8.8.8.8",
			"8.8.4.4",
			"1.1.1.1",
			"192.168.0.0/16",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"224.0.0.0/4", // multicast — SSDP, mDNS, BitTorrent LPD
		},
	}
}
