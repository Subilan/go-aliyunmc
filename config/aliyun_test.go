package config

import "testing"

func TestAliyunEndpoints(t *testing.T) {
	cfg := Aliyun{RegionId: "cn-hangzhou"}

	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{name: "ECS Endpoint", endpoint: cfg.EcsEndpoint(), expected: "ecs.cn-hangzhou.aliyuncs.com"},
		{name: "OSS Endpoint", endpoint: cfg.OssEndpoint(), expected: "oss-cn-hangzhou.aliyuncs.com"},
		{name: "VPC Endpoint", endpoint: cfg.VpcEndpoint(), expected: "vpc.cn-hangzhou.aliyuncs.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.endpoint != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.endpoint)
			}
		})
	}
}
