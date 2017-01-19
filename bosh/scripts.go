package bosh

import (
	"path/filepath"
)

type BackupAndRestoreScripts []string

const (
	backupScriptName  = "p-backup"
	restoreScriptName = "p-restore"

	jobDirectoryMatcher  = "/var/vcap/jobs/*/bin/"
	backupScriptMatcher  = jobDirectoryMatcher + backupScriptName
	restoreScriptMatcher = jobDirectoryMatcher + restoreScriptName
)

func NewBackupAndRestoreScripts(files []string) BackupAndRestoreScripts {
	bandrScripts := []string{}
	for _, script := range files {
		if match, _ := filepath.Match(backupScriptMatcher, script); match {
			bandrScripts = append(bandrScripts, script)
		}
		if match, _ := filepath.Match(restoreScriptMatcher, script); match {
			bandrScripts = append(bandrScripts, script)
		}
	}
	return bandrScripts
}
