package config

type Config struct {
	Logging		LoggingConfig		`toml:"logging" json:"logging"`
	Server		Server		`toml:"server" json:"server"`
}

type Server struct {
	Bind	string			`toml:"bind" json:"bind"`
	Balance	string			`toml:"balance" json:"balance"`
	Discovery *DiscoveryConfig	`toml:"discovery" json:"discovery"`
	Healthcheck *HealthcheckConfig	`toml:"healthcheck" json:"healthcheck"`
}

type LoggingConfig struct {
	Level	string			`toml:"level" json:"level"`
	Output	string			`toml:"output" json:"output"`
}

type DiscoveryConfig struct {
	Kind	string			`toml:"kind" json:"kind"`
	*StaticDiscoveryConfig
}

type StaticDiscoveryConfig struct {
	StaticList []string	`toml:"static_list" json:"static_list"`
}

type HealthcheckConfig struct {
	Kind     string `toml:"kind" json:"kind"`
	Interval string `toml:"interval" json:"interval"`
	Timeout  string `toml:"timeout" json:"timeout"`
	Count   int    `toml:"count" json:"count"`
	Loss   float64    `toml:"loss" json:"loss"`
	Rtt  string `toml:"rtt" json:"rtt"`

	/* Depends on Kind */

	*PingHealthcheckConfig
}

type PingHealthcheckConfig struct {
}

