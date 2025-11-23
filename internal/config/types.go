package configuration

import (
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

// Config defines the interface for retrieving configuration data.
type Config interface {
	GetAllDataSources() map[string][]*bloopi_agent.DataSource
	GetCoordimapKey() (string, error)
	GetSkipFields() []string
}

// BloopiConfigNameValueConfig represents a name-value pair configuration item.
type BloopiConfigNameValueConfig struct {
	Name  string
	Value string
	Send  bool
}

// BloopiConfigDataSource represents a data source configuration.
type BloopiConfigDataSource struct {
	Type   string
	Name   string
	Desc   string
	ID     string
	Config []BloopiConfigNameValueConfig
}

// Coordimap holds the configuration specific to the Coordimap integration.
type Coordimap struct {
	APIKey      string                   `yaml:"api_key"`
	SkipFields  []string                 `yaml:"skip_fields"`
	DataSources []BloopiConfigDataSource `yaml:"data_sources"`
}

// CoordimapConfig represents the top-level configuration structure for Coordimap.
type CoordimapConfig struct {
	Coordimap Coordimap `yaml:"coordimap"`
}
