package aliyun

import (
	"testing"
)

func TestAliyunConfig_Endpoints(t *testing.T) {
	config := Config{
		RegionId: "cn-hangzhou",
	}

	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "ECS Endpoint",
			endpoint: config.EcsEndpoint(),
			expected: "ecs.cn-hangzhou.aliyuncs.com",
		},
		{
			name:     "OSS Endpoint",
			endpoint: config.OssEndpoint(),
			expected: "oss-cn-hangzhou.aliyuncs.com",
		},
		{
			name:     "VPC Endpoint",
			endpoint: config.VpcEndpoint(),
			expected: "vpc.cn-hangzhou.aliyuncs.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.endpoint != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.endpoint)
			}
		})
	}
}
