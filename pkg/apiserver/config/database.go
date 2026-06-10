package config

import "time"

// DatabaseSettings holds the settings for database connections.
type DatabaseSettings struct {
	Type           DatabaseType
	Endpoints      []string
	ConnectTimeout time.Duration
	DatabaseName   string

	DDLAuto bool
}

// DatabaseType represents the type of database to be used.
type DatabaseType string

const (
	// DatabaseTypeMongoDB represents a MongoDB database.
	DatabaseTypeMongoDB DatabaseType = "mongodb"
	// DatabaseTypeInMemory represents an in-memory database used for standalone
	// (single-node, no external dependency) mode. Data is not persisted across
	// restarts.
	DatabaseTypeInMemory DatabaseType = "inmemory"
)
