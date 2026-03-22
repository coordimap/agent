package configuration_test

import (
	configuration "coordimap-agent/internal/config"
	"os"
	"reflect"
	"testing"
)

func TestNewYamlFileConfig(t *testing.T) {
	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := []byte(`
coordimap:
  api_key: test_key
  data_sources: []
`)
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid file",
			args:    args{filePath: tmpfile.Name()},
			wantErr: false,
		},
		{
			name:    "non-existent file",
			args:    args{filePath: "non_existent_file.yaml"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := configuration.NewYamlFileConfig(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewYamlFileConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
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
			name: "valid config",
			want: &configuration.CoordimapConfig{
				Coordimap: configuration.Coordimap{
					APIKey: "123",
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
			args: args{yamlContent: `coordimap:
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
		{
			name:    "missing api key",
			want:    nil,
			wantErr: true,
			args: args{yamlContent: `coordimap:
  data_sources: []`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configuration.NewYamlStringConfig(tt.args.yamlContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewYamlStringConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewYamlStringConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
