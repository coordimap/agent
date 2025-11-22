package configuration_test

import (
	"cleye/internal/config"
	"testing"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

func TestBloopiConfig_GetAllDataSources(t *testing.T) {
	tests := []struct {
		stringConfig []byte
		name         string
		want         map[string]*bloopi_agent.DataSource
	}{
		{
			name: "test1",
			stringConfig: []byte(`
bloopi:
    api_key: ${BLOOPI_API_KEY}
data_sources:
  aws:
    info:
        name: aws1
        desc: desc1
    config:
        policy_config: "true"
        access_key_id: "${ACCESS_KEY_ID}"
        secret_access_key: "${SECRET_ACCSS_KEY}"
  
  postgres:
    info:
        name: post1
        desc: desc1
    config:
        db_name: dbname1
        db_user: user1
        db_pass: pass1
        db_host: host1`),
			want: map[string]*bloopi_agent.DataSource{
				"aws": {
					Info: bloopi_agent.DataSourceInfo{
						Name: "aws1",
						Desc: "desc1",
						Type: "aws",
					},
					Config: bloopi_agent.DataSourceConfig{
						ValuePairs: []bloopi_agent.KeyValue{
							{
								Key:   "policy_config",
								Value: "true",
							},
							{
								Key:   "access_key_id",
								Value: "",
							},
							{
								Key:   "secret_access_key",
								Value: "",
							},
						},
					},
				},
				"postgres": {
					Info: bloopi_agent.DataSourceInfo{
						Name: "post1",
						Desc: "desc1",
						Type: "postgres",
					},
					Config: bloopi_agent.DataSourceConfig{
						ValuePairs: []bloopi_agent.KeyValue{
							{
								Key:   "db_name",
								Value: "dbname1",
							},
							{
								Key:   "db_user",
								Value: "user1",
							},
							{
								Key:   "db_pass",
								Value: "pass1",
							},
							{
								Key:   "db_host",
								Value: "host1",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := configuration.OldStringConfig(tt.stringConfig)
			configDSs := config.GetAllDataSources()

			for dsName, ds := range configDSs {
				if _, exists := tt.want[dsName]; !exists {
					t.Errorf("the DataSource: %s cannot be found in the tests", dsName)
				}

				testDS := tt.want[dsName]
				if testDS.Info.Name != ds.Info.Name || testDS.Info.Desc != ds.Info.Desc || testDS.Info.Type != ds.Info.Type {
					t.Errorf("the config ds info: %v is not the same as the test ds info: %v", testDS.Info, ds.Info)
				}

				for _, valuePair := range ds.Config.ValuePairs {
					valuePairFound := false

					for _, testValuePair := range testDS.Config.ValuePairs {
						if valuePair.Key == testValuePair.Key && valuePair.Value == testValuePair.Value {
							valuePairFound = true
							break
						}
					}

					if !valuePairFound {
						t.Errorf("the config ValuePair: %v was not found in the tests", valuePair)
						break
					}
				}
			}
		})
	}
}
