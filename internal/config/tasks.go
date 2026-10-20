package config

// Tasks 对应 [tasks]。
type Tasks struct {
	Deploy  TasksDeploy  `toml:"deploy"`
	Backup  TasksBackup  `toml:"backup"`
	Archive TasksArchive `toml:"archive"`
}

// TasksDeployVars 对应 [tasks.deploy.vars]。
type TasksDeployVars struct {
	SSHAccessPublicKey string `toml:"ssh_access_public_key,commented" validate:"required" comment:"用于SSH访问的公钥"`

	JavaVersion int `toml:"java_version" default:"21" validate:"min=1" comment:"JRE版本号，建议使用当前服务端要求的最低版本"`

	Packages []string `toml:"packages" validate:"dive,min=1" comment:"额外安装的软件包列表，例如 ['zip', 'unzip']"`

	ArchiveOSSPath string `toml:"archive_oss_path,commented" validate:"required" comment:"用于部署恢复的OSS归档路径"`
	WssCertSrc     string `toml:"wss_cert_src,commented" comment:"WSS证书文件路径（.p12格式），留空则跳过证书上传"`
	WssCertDst     string `toml:"wss_cert_dst,commented" comment:"WSS证书在远程服务器上的目标路径，留空则跳过证书上传"`
}

// TasksDeploy 对应 [tasks.deploy]。
type TasksDeploy struct {
	Vars TasksDeployVars `toml:"vars" comment:"部署脚本模板变量"`
}

// TasksBackupVars 对应 [tasks.backup.vars]。
type TasksBackupVars struct {
	BackupOSSPath string `toml:"backup_oss_path,commented" validate:"required" comment:"备份上传目标OSS目录"`

	MaxKeepCount int `toml:"max_keep_count" default:"5" validate:"min=1" comment:"备份保留数量上限，超过后删除最旧的备份"`

	BaseDir string `toml:"base_dir" default:"/home/mc/server/archive" comment:"需要备份的目录的公共父目录"`

	TargetDirs []string `toml:"target_dirs" validate:"required,min=1,dive,min=1" comment:"需要备份的目录列表（相对于BaseDir），例如 ['world', 'world_nether', 'world_the_end']"`
}

// TasksBackup 对应 [tasks.backup]。
type TasksBackup struct {
	TemplatePath string `toml:"template_path" default:"assets/scripts/backup.tmpl.sh" comment:"备份脚本模板路径"`

	TimeoutSec int `toml:"timeout_sec" default:"1800" validate:"min=1" comment:"任务超时时间（秒）"`

	Exclusive bool `toml:"exclusive" comment:"是否同类互斥执行"`

	Vars TasksBackupVars `toml:"vars" comment:"备份脚本模板变量"`
}

// TasksArchiveVars 对应 [tasks.archive.vars]。
type TasksArchiveVars struct {
	OSSRoot          string `toml:"oss_root,commented" validate:"required,min=1" comment:"归档OSS根路径"`
	ArchiveName      string `toml:"archive_name" default:"archive" validate:"min=1" comment:"归档目录名称"`
	RemoteArchiveDir string `toml:"remote_archive_dir" default:"/home/mc/server/archive" validate:"min=1" comment:"位于远程服务器上的归档目录的绝对路径"`
}

// TasksArchive 对应 [tasks.archive]。
type TasksArchive struct {
	TemplatePath string `toml:"template_path" default:"assets/scripts/archive.tmpl.sh" comment:"归档脚本模板路径"`

	TimeoutSec int `toml:"timeout_sec" default:"600" validate:"min=1" comment:"任务超时时间（秒）"`

	Vars TasksArchiveVars `toml:"vars" comment:"归档脚本模板变量"`
}
