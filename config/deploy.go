package config

import "path/filepath"

type DeployConfig struct {
	Packages     []string `toml:"packages" validate:"required"`
	SSHPublicKey string   `toml:"ssh_public_key" validate:"required"`
	JavaVersion  uint     `toml:"java_version" validate:"required,min=8"`
	OSSRoot      string   `toml:"oss_root" validate:"required"`
	BackupPath   string   `toml:"backup_path" validate:"required"`
	ArchivePath  string   `toml:"archive_path" validate:"required"`
}

func (d DeployConfig) BackupOSSPath() string {
	return "oss://" + filepath.Join(d.OSSRoot[6:], d.BackupPath)
}

func (d DeployConfig) ArchiveOSSPath() string {
	return "oss://" + filepath.Join(d.OSSRoot[6:], d.ArchivePath)
}
