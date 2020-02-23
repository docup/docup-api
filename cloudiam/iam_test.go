package cloudiam

import "testing"

func TestGetIAMSample(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "hoge",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetIAMSample()
		})
	}
}

func TestGetIAMPolicySample(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "expt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetIAMPolicySample()
		})
	}
}
