package utils_test

import (
	"cleye/utils"
	"os"
	"testing"
)

func TestLoadValueFromEnvConfig(t *testing.T) {
	type args struct {
		value string
		env   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "test1",
			args:    args{value: "${TEST_ENV1}", env: "TEST_ENV1"},
			want:    "123",
			wantErr: false,
		},
		{
			name:    "test2",
			args:    args{value: "${TEST_ENV2}", env: "TEST_ENV1"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "test3",
			args:    args{value: "${TEST_ENV2", env: "TEST_ENV1"},
			want:    "${TEST_ENV2",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.args.env, tt.want)
			got, err := utils.LoadValueFromEnvConfig(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadValueFromEnvConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LoadValueFromEnvConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
