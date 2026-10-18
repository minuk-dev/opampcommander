package config

// OpAMPSettings controls OpAMP transport message limits and compression.
// Its zero value keeps the opamp-go default limit and disables compression.
type OpAMPSettings struct {
	MaxMessageSize    int64 `mapstructure:"maxMessageSize"`
	EnableCompression bool  `mapstructure:"enableCompression"`
}

// DefaultOpAMPSettings preserves the default 64 MiB limit without compression.
func DefaultOpAMPSettings() OpAMPSettings {
	return OpAMPSettings{MaxMessageSize: 0, EnableCompression: false}
}

// Validate accepts every int64: zero selects the library default, negative
// values disable the limit, and positive values specify a byte limit.
func (s OpAMPSettings) Validate() error {
	return nil
}
