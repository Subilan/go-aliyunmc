package tasks

import "embed"

//go:embed scripts/*.tmpl.sh
var scriptTemplates embed.FS

const (
	scriptDir           = "scripts/"
	backupTemplateName  = scriptDir + "backup.tmpl.sh"
	archiveTemplateName = scriptDir + "archive.tmpl.sh"
)
