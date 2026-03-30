package aliyun

// MustInitialize 初始化各个阿里云client
func MustInitialize() {
	var err error

	OssClient = GetOssClient()
	EcsClient, err = ShouldCreateEcsClient()
	if err != nil {
		panic(err)
	}
	VpcClient, err = ShouldCreateVpcClient()
	if err != nil {
		panic(err)
	}
	BssClient, err = ShouldCreateBssClient()
	if err != nil {
		panic(err)
	}
}
