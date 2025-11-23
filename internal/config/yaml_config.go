package configuration

import (
	"cleye/pkg/utils"
	"fmt"
	"os"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"gopkg.in/yaml.v3"
)

type yamlConfig struct {
	parsedConfig   *Coordimap
	yamlConfigPath string
}

func (coordimapConfig *yamlConfig) GetCoordimapKey() (string, error) {
	if coordimapConfig.parsedConfig == nil {
		return "", fmt.Errorf("configuration is nil")
	}

	value, err := utils.LoadValueFromEnvConfig(coordimapConfig.parsedConfig.API_KEY)
	if err != nil {
		return "", err
	}

	return value, nil
}

func (coordimapConfig *yamlConfig) GetSkipFields() []string {
	if coordimapConfig.parsedConfig == nil {
		return []string{}
	}

	return coordimapConfig.parsedConfig.SkipFields
}

func (coordimapConfig *yamlConfig) GetAllDataSources() map[string][]*bloopi_agent.DataSource {
	if coordimapConfig.parsedConfig == nil {
		return map[string][]*bloopi_agent.DataSource{}
	}

	allDataSources := map[string][]*bloopi_agent.DataSource{}
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
			Info:         info,
			DataSourceID: dataSource.ID,
			Config: bloopi_agent.DataSourceConfig{
				ValuePairs: dsValuePairs,
			},
		}

		allDataSources[info.Type] = append(allDataSources[info.Type], currentDS)
	}

	return allDataSources
}

// NewYamlFileConfig reads in the yaml file provided in the path and generates the correct config structure
func NewYamlFileConfig(filePath string) (Config, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	parsedYaml, errParsedYaml := NewYamlStringConfig(string(yamlFile))
	if errParsedYaml != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", errParsedYaml)
	}

	return &yamlConfig{
		parsedConfig:   &parsedYaml.Coordimap,
		yamlConfigPath: filePath,
	}, nil
}

func NewYamlStringConfig(yamlContent string) (*CoordimapConfig, error) {
	config := CoordimapConfig{}

	if errorUnmarshal := yaml.Unmarshal([]byte(yamlContent), &config); errorUnmarshal != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", errorUnmarshal)
	}

	// Basic validation
	if config.Coordimap.API_KEY == "" {
		// Check if it's an env var placeholder, if not, it's missing
		// Actually, even if it is a placeholder, it should be present in the struct.
		// If the string is empty, it means the key is missing from YAML.
		return nil, fmt.Errorf("missing required field: coordimap.api_key")
	}

	return &config, nil
}
