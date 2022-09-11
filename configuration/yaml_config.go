package configuration

import (
	"cleye/utils"
	"fmt"
	"os"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"gopkg.in/yaml.v3"
)

type yamlConfig struct {
	parsedConfig    *Coordimap
	parsedCorrectly bool
	yamlConfigPath  string
}

func (coordimapConfig *yamlConfig) GetCoordimapKey() (string, error) {
	if !coordimapConfig.parsedCorrectly {
		return "", fmt.Errorf("could not parse successfully the file at: %s", coordimapConfig.yamlConfigPath)
	}

	value, err := utils.LoadValueFromEnvConfig(coordimapConfig.parsedConfig.API_KEY)
	if err != nil {
		return "", err
	}

	return value, nil
}

func (coordimapConfig *yamlConfig) GetAllDataSources() map[string]*bloopi_agent.DataSource {
	if !coordimapConfig.parsedCorrectly {
		return map[string]*bloopi_agent.DataSource{}
	}

	allDataSources := map[string]*bloopi_agent.DataSource{}
	for _, dataSource := range coordimapConfig.parsedConfig.DataSources {
		info := bloopi_agent.DataSourceInfo{
			Name: dataSource.Name,
			Desc: dataSource.Desc,
			Type: dataSource.Type,
		}

		dsValuePairs := []bloopi_agent.KeyValue{}
		for _, valuePair := range dataSource.Config {
			dsValuePairs = append(dsValuePairs, bloopi_agent.KeyValue{
				Key:   valuePair.Name,
				Value: valuePair.Value,
			})
		}

		currentDS := &bloopi_agent.DataSource{
			Info: info,
			Config: bloopi_agent.DataSourceConfig{
				ValuePairs: dsValuePairs,
			},
		}

		allDataSources[info.Type] = currentDS
	}

	return allDataSources
}

// NewYamlFileConfig reads in the yaml file provided in the path and generates the correct config structure
func NewYamlFileConfig(filePath string) (Config, error) {
	yamlFile, err := os.ReadFile(filePath)

	if err != nil {
		return nil, err
	}

	parsedYaml, errParsedYaml := NewYamlStringConfig(string(yamlFile))
	if errParsedYaml != nil {
		return &yamlConfig{
			parsedConfig:    nil,
			parsedCorrectly: false,
			yamlConfigPath:  filePath,
		}, err
	}

	return &yamlConfig{
		parsedConfig:    &parsedYaml.Coordimap,
		parsedCorrectly: true,
		yamlConfigPath:  filePath,
	}, nil
}

func NewYamlStringConfig(yamlContent string) (*CoordimapConfig, error) {
	config := CoordimapConfig{}

	if errorUnmarshal := yaml.Unmarshal([]byte(yamlContent), &config); errorUnmarshal != nil {
		return nil, errorUnmarshal
	}

	return &config, nil
}
