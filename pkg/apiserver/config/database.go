package config

import "time"

// DatabaseSettings holds the settings for database connections.
type DatabaseSettings struct {
	Type           DatabaseType
	Endpoints      []string
	ConnectTimeout time.Duration
}

// DatabaseType represents the type of database to be used.
type DatabaseType string

const (
	// DatabaseTypeEtcd represents an etcd database.
	DatabaseTypeEtcd DatabaseType = "etcd"
	// DatabaseTypeMongoDB represents a MongoDB database.
	DatabaseTypeMongoDB DatabaseType = "mongodb"
)
