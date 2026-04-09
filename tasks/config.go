package tasks

import "go-aliyunmc/utils"

type DeployTemplateVars struct {
	SSHPublicKey   string   `toml:"ssh_public_key" validate:"required" comment:"部署用户写入的公钥"`
	JavaVersion    int      `toml:"java_version" validate:"required,min=1" comment:"Zulu Java主版本号"`
	Packages       []string `toml:"packages" validate:"omitempty,dive,min=1" comment:"额外安装的软件包列表"`
	ArchiveOSSPath string   `toml:"archive_oss_path" validate:"required" comment:"用于部署恢复的OSS归档路径"`
}

type DeployStepConfig struct {
	Name       string `toml:"name" validate:"required" comment:"步骤名称"`
	ScriptPath string `toml:"script_path" validate:"required" comment:"步骤脚本模板路径"`
	TimeoutSec int    `toml:"timeout_sec" validate:"required,min=1" comment:"步骤执行超时时间（秒）"`
}

type DeployTaskConfig struct {
	Exclusive bool               `toml:"exclusive" comment:"是否同类互斥执行"`
	Steps     []DeployStepConfig `toml:"steps" validate:"required,min=1,dive" comment:"按顺序执行的部署步骤列表"`
	Vars      DeployTemplateVars `toml:"vars" validate:"required" comment:"部署脚本模板变量"`
}

type BackupTemplateVars struct {
	BackupOSSPath string `toml:"backup_oss_path" validate:"required" comment:"备份上传目标OSS目录"`
}

type BackupTaskConfig struct {
	TemplatePath string             `toml:"template_path" validate:"required" comment:"备份脚本模板路径"`
	TimeoutSec   int                `toml:"timeout_sec" validate:"required,min=1" comment:"任务超时时间（秒）"`
	Exclusive    bool               `toml:"exclusive" comment:"是否同类互斥执行"`
	Vars         BackupTemplateVars `toml:"vars" validate:"required" comment:"备份脚本模板变量"`
}

type ArchiveTemplateVars struct {
	OSSRoot string `toml:"oss_root" validate:"required" comment:"归档轮转根路径"`
}

type ArchiveTaskConfig struct {
	TemplatePath string              `toml:"template_path" validate:"required" comment:"归档脚本模板路径"`
	TimeoutSec   int                 `toml:"timeout_sec" validate:"required,min=1" comment:"任务超时时间（秒）"`
	Exclusive    bool                `toml:"exclusive" comment:"是否同类互斥执行"`
	Vars         ArchiveTemplateVars `toml:"vars" validate:"required" comment:"归档脚本模板变量"`
}

var DeployC DeployTaskConfig
var BackupC BackupTaskConfig
var ArchiveC ArchiveTaskConfig

func MustLoadConfig() {
	utils.MustBindConfig(&DeployC, "task-deploy")
	utils.MustBindConfig(&BackupC, "task-backup")
	utils.MustBindConfig(&ArchiveC, "task-archive")
}
