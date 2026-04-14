package db

import "database/sql"

type DBStats struct {
	Open  int   `json:"open"`
	InUse int   `json:"in_use"`
	Idle  int   `json:"idle"`
	Wait  int64 `json:"wait"`
}

func GetStats(sqlDB *sql.DB) DBStats {
	stats := sqlDB.Stats()
	return DBStats{
		Open:  stats.OpenConnections,
		InUse: stats.InUse,
		Idle:  stats.Idle,
		Wait:  stats.WaitCount,
	}
}
