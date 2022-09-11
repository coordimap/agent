package configuration_test

import (
	"cleye/configuration"
	"reflect"
	"testing"
)

func TestNewYamlFileConfig(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    *configuration.CoordimapConfig
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configuration.NewYamlFileConfig(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewYamlFileConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewYamlFileConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewYamlStringConfig(t *testing.T) {
	type args struct {
		yamlContent string
	}
	tests := []struct {
		name    string
		args    args
		want    *configuration.CoordimapConfig
		wantErr bool
	}{
		{
			name: "string 1",
			want: &configuration.CoordimapConfig{
				Coordimap: configuration.Coordimap{
					API_KEY: "123",
					DataSources: []configuration.BloopiConfigDataSource{
						{
							Type: "aws",
							Name: "aws1",
							Desc: "desc1",
							Config: []configuration.BloopiConfigNameValueConfig{
								{
									Name:  "policy_config",
									Value: "true",
								},
							},
						},
						{
							Type: "postgres",
							Name: "post1",
							Desc: "desc1",
							Config: []configuration.BloopiConfigNameValueConfig{
								{
									Name:  "db_name",
									Value: "dbname1",
								},
								{
									Name:  "db_host",
									Value: "host1",
								},
								{
									Name:  "db_user",
									Value: "user1",
								},
								{
									Name:  "db_pass",
									Value: "pass1",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			args: args{yamlContent: `bloopi:
  api_key: 123
  data_sources:
    - type: aws
      name: aws1
      desc: desc1
      config:
      - name: policy_config
        value: "true"
    - type: postgres
      name: post1
      desc: desc1
      config:
        - name: db_name
          value: dbname1
        - name: db_host
          value: host1
        - name: db_user
          value: user1
        - name: db_pass
          value: pass1`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configuration.NewYamlStringConfig(tt.args.yamlContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewYamlStringConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewYamlStringConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
