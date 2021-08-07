package configuration

import (
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/spf13/viper"
)

type Config interface {
	GetAllDataSources() map[string]*bloopi_agent.DataSource
	GetBloopiKey() (string, error)
}

type OldBloopiConfig struct {
	viper *viper.Viper
}

type BloopiConfigNameValueConfig struct {
	Name  string
	Value string
	Send  bool
}

type BloopiConfigDataSource struct {
	Type   string
	Name   string
	Desc   string
	Config []BloopiConfigNameValueConfig
}

type Bloopi struct {
	API_KEY     string                   `yaml:"api_key"`
	DataSources []BloopiConfigDataSource `yaml:"data_sources"`
}

type BloopiConfig struct {
	Bloopi Bloopi `yaml:"bloopi"`
}
