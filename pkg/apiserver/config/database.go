package config

import "time"

// DatabaseSettings holds the settings for database connections.
type DatabaseSettings struct {
	Type           DatabaseType
	Endpoints      []string
	ConnectTimeout time.Duration
	DatabaseName   string

	DDLAuto  bool
	Sharding ShardingSettings
}

// ShardingSettings turns on sharding-aware schema management for a MongoDB
// sharded cluster (i.e. when Endpoints point at mongos routers).
type ShardingSettings struct {
	// Enabled makes EnsureSchema enable sharding on the database and run
	// shardCollection for the sharded collections using the built-in shard-key
	// plan. It is a no-op unless DDLAuto is also set.
	Enabled bool
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
