package templateData

import "github.com/Subilan/go-aliyunmc/config"

type DeployTemplateData struct {
	Username        string
	Password        string
	SSHPublicKey    string
	Packages        []string
	RegionId        string
	AccessKeyId     string
	AccessKeySecret string
	JavaVersion     uint
	DataDiskSize    int
	ArchiveOSSPath  string
}

func Deploy() DeployTemplateData {
	return DeployTemplateData{
		Username:        "mc",
		Password:        config.Cfg.Aliyun.Ecs.ProdPassword,
		SSHPublicKey:    config.Cfg.Deploy.SSHPublicKey,
		Packages:        config.Cfg.Deploy.Packages,
		RegionId:        config.Cfg.Aliyun.RegionId,
		AccessKeyId:     config.Cfg.Aliyun.AccessKeyId,
		AccessKeySecret: config.Cfg.Aliyun.AccessKeySecret,
		JavaVersion:     config.Cfg.Deploy.JavaVersion,
		DataDiskSize:    config.Cfg.Aliyun.Ecs.DataDisk.Size,
		ArchiveOSSPath:  config.Cfg.Deploy.ArchiveOSSPath,
	}
}
