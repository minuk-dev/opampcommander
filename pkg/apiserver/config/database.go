package config

// DatabaseSettings holds the settings for database connections.
type DatabaseSettings struct {
	Type      DatabaseType
	Endpoints []string
}

// DatabaseType represents the type of database to be used.
type DatabaseType string

const (
	// DatabaseTypeEtcd represents an etcd database.
	DatabaseTypeEtcd DatabaseType = "etcd"
)
