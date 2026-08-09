package tasks

import "github.com/Subilan/go-aliyunmc/utils"

type DeployTemplateVars struct {
	SSHAccessPublicKey string   `toml:"ssh_access_public_key" validate:"required" comment:"用于SSH访问的公钥"`
	SSHGomcPublicKey   string   `toml:"ssh_gomc_public_key" validate:"required" comment:"gomc服务所在主机的公钥"`
	JavaVersion        int      `toml:"java_version" validate:"required,min=1" comment:"JRE版本号，建议使用当前服务端要求的最低版本"`
	Packages           []string `toml:"packages" validate:"omitempty,dive,min=1" comment:"额外安装的软件包列表"`
	ArchiveOSSPath     string   `toml:"archive_oss_path" validate:"required" comment:"用于部署恢复的OSS归档路径"`
	WssCertSrc         string   `toml:"wss_cert_src" comment:"WSS证书文件路径（.p12格式），留空则跳过证书上传"`
	WssCertDst         string   `toml:"wss_cert_dst" comment:"WSS证书在远程服务器上的目标路径，留空则跳过证书上传"`
}

type DeployTaskConfig struct {
	Vars DeployTemplateVars `toml:"vars" validate:"required" comment:"部署脚本模板变量"`
}

type BackupTemplateVars struct {
	BackupOSSPath string   `toml:"backup_oss_path" validate:"required" comment:"备份上传目标OSS目录"`
	MaxKeepCount  int      `toml:"max_keep_count" validate:"required,min=1" comment:"备份保留数量上限，超过后删除最旧的备份"`
	BaseDir       string   `toml:"base_dir" validate:"required" comment:"需要备份的目录的公共父目录"`
	TargetDirs    []string `toml:"target_dirs" validate:"required,min=1,dive,min=1" comment:"需要备份的目录列表（相对于BaseDir）"`
}

type BackupTaskConfig struct {
	TemplatePath string             `toml:"template_path" validate:"required" comment:"备份脚本模板路径"`
	TimeoutSec   int                `toml:"timeout_sec" validate:"required,min=1" comment:"任务超时时间（秒）"`
	Exclusive    bool               `toml:"exclusive" comment:"是否同类互斥执行"`
	Vars         BackupTemplateVars `toml:"vars" validate:"required" comment:"备份脚本模板变量"`
}

type ArchiveTemplateVars struct {
	OSSRoot          string `toml:"oss_root" validate:"required,min=1" comment:"归档OSS根路径"`
	ArchiveName      string `toml:"archive_name" validate:"required,min=1" comment:"归档目录名称"`
	RemoteArchiveDir string `toml:"remote_archive_dir" validate:"required,min=1" comment:"位于远程服务器上的归档目录的绝对路径"`
}

type ArchiveTaskConfig struct {
	TemplatePath string              `toml:"template_path" validate:"required" comment:"归档脚本模板路径"`
	TimeoutSec   int                 `toml:"timeout_sec" validate:"required,min=1" comment:"任务超时时间（秒）"`
	Vars         ArchiveTemplateVars `toml:"vars" validate:"required" comment:"归档脚本模板变量"`
}

var DeployC DeployTaskConfig
var BackupC BackupTaskConfig
var ArchiveC ArchiveTaskConfig

func MustLoadConfig() {
	utils.MustBindConfigToml(&DeployC, "task-deploy")
	utils.MustBindConfigToml(&BackupC, "task-backup")
	utils.MustBindConfigToml(&ArchiveC, "task-archive")
}
