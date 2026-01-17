package config

import "path/filepath"

// DeployConfig 包含与实例部署行为的相关配置内容。目前实例部署仅支持 debian 系系统，欢迎贡献扩展。
type DeployConfig struct {
	// Packages 包含了在实例上需要安装的包名称（不包含 Java）
	Packages []string `toml:"packages" validate:"required" comment:"部署阶段需要安装的包名称，注意拼写正确，不包含Java"`

	// SSHPublicKey 是服务器管理者个人持有的公钥，用于快速登录服务器
	SSHPublicKey string `toml:"ssh_public_key" validate:"required" comment:"服务器管理者个人公钥，用于快速登录"`

	// JavaVersion 是实例所运行的 Java 版本，最低为 8
	JavaVersion uint `toml:"java_version" validate:"required,min=8" comment:"需要预装的Java版本，最低为8"`

	// OSSRoot 是用于存储归档和备份信息的存储桶地址，必须以 oss:// 开头
	OSSRoot string `toml:"oss_root" validate:"required" comment:"用于存储归档和备份的OSS存储桶地址，必须以oss://开头"`

	// BackupPath 是用于存放备份的存储桶内地址，相对于 OSSRoot
	BackupPath string `toml:"backup_path" validate:"required" comment:"用于存储备份的存储桶内地址，相对于OSSRoot，例如/backups"`

	// ArchivePath 是用于存储归档的存储桶内地址，相对于 OSSRoot
	ArchivePath string `toml:"archive_path" validate:"required" comment:"用于存储归档的备份桶内地址，相对于OSSRoot，例如/archive"`
}

// BackupOSSPath 返回从 OSSRoot 和 BackupPath 合并出的最终存储桶内地址
func (d DeployConfig) BackupOSSPath() string {
	return "oss://" + filepath.Join(d.OSSRoot[6:], d.BackupPath)
}

// ArchiveOSSPath 返回从 OSSRoot 和 ArchivePath 合并出的最终存储桶内地址
func (d DeployConfig) ArchiveOSSPath() string {
	return "oss://" + filepath.Join(d.OSSRoot[6:], d.ArchivePath)
}
