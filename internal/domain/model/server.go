package model

import "time"

// Server represents a server that an agent communicates with.
// It contains the server's unique identifier and liveness information.
type Server struct {
	// ID is the unique identifier for the server.
	ID string
	// LastHeartbeatAt is the last time the server sent a heartbeat.
	LastHeartbeatAt time.Time
	// CreatedAt is the time the server was first registered.
	CreatedAt time.Time
}

// IsAlive returns true if the server is alive based on the heartbeat timeout.
// A server is considered alive if its last heartbeat was within the timeout period.
func (s *Server) IsAlive(now time.Time, timeout time.Duration) bool {
	return now.Sub(s.LastHeartbeatAt) < timeout
}
