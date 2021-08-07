package configuration

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/spf13/viper"
)

func OldFileConfig(filePath string) *OldBloopiConfig {
	v := viper.GetViper()
	v.SetConfigFile(filePath)
	v.SetConfigType("yaml")

	errReadConfig := v.ReadInConfig()
	if errReadConfig != nil {
		fmt.Println(errReadConfig)
		panic("error")
	}

	fmt.Println(v.GetString("bloopi.api_key"))

	return &OldBloopiConfig{
		viper: v,
	}
}

func OldStringConfig(yamlConfig []byte) *OldBloopiConfig {
	v := viper.GetViper()
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(yamlConfig))

	return &OldBloopiConfig{
		viper: v,
	}
}

func (config *OldBloopiConfig) evaluateValue(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		// Evaluate Env Variable
		if envVal, ok := os.LookupEnv(value[2 : len(value)-1]); ok {
			return envVal
		}

		return ""
	}

	return value
}

func (config *OldBloopiConfig) getAllFirstLevelKeys(v *viper.Viper) []string {
	firstLevelKeys := []string{}
	foundKeys := map[string]string{}
	allKeys := v.AllKeys()

	for _, key := range allKeys {
		firstLevelKey := key[0:strings.Index(key, ".")]
		if _, exists := foundKeys[firstLevelKey]; !exists {
			foundKeys[firstLevelKey] = ""
			firstLevelKeys = append(firstLevelKeys, firstLevelKey)
		}
	}

	return firstLevelKeys
}

func (config *OldBloopiConfig) GetAllDataSources() map[string]*bloopi_agent.DataSource {
	allDSs := map[string]*bloopi_agent.DataSource{}

	configuredDataSources := config.viper.Sub("data_sources")
	allDataSourceNames := config.getAllFirstLevelKeys(configuredDataSources)

	for _, dataSourceName := range allDataSourceNames {
		info := configuredDataSources.GetStringMapString(fmt.Sprintf("%s.info", dataSourceName))
		dsInfo := bloopi_agent.DataSourceInfo{
			Name: info["name"],
			Desc: info["desc"],
			Type: dataSourceName,
		}

		valuePairs := []bloopi_agent.KeyValue{}
		for key, value := range configuredDataSources.GetStringMapString(fmt.Sprintf("%s.config", dataSourceName)) {
			valuePairs = append(valuePairs, bloopi_agent.KeyValue{
				Key:   key,
				Value: config.evaluateValue(value),
			})
		}

		dsConfig := bloopi_agent.DataSourceConfig{
			ValuePairs: valuePairs,
		}

		allDSs[dataSourceName] = &bloopi_agent.DataSource{
			Info:   dsInfo,
			Config: dsConfig,
		}
	}

	return allDSs
}

func (config *OldBloopiConfig) GetBloopiKey() (string, error) {
	if !config.viper.IsSet("bloopi.api_key") {
		return "", errors.New("the key bloopi.api_key must be set")
	}

	return config.viper.GetString("bloopi.api_key"), nil
}
