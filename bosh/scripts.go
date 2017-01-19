package bosh

import (
	"path/filepath"
)

type BackupAndRestoreScripts []script

const (
	backupScriptName  = "p-backup"
	restoreScriptName = "p-restore"

	jobDirectoryMatcher  = "/var/vcap/jobs/*/bin/"
	backupScriptMatcher  = jobDirectoryMatcher + backupScriptName
	restoreScriptMatcher = jobDirectoryMatcher + restoreScriptName
)

type script string

func (s script) isBackup() bool {
	match, _ := filepath.Match(backupScriptMatcher, string(s))
	return match
}

func (s script) isRestore() bool {
	match, _ := filepath.Match(restoreScriptMatcher, string(s))
	return match
}

func (s script) isPlatformScript() bool {
	return s.isBackup() || s.isRestore()
}


func NewBackupAndRestoreScripts(files []string) BackupAndRestoreScripts {
	bandrScripts := []script{}
	for _, s := range files {
		s:=script(s)
		if s.isPlatformScript(){
			bandrScripts = append(bandrScripts, s)
		}
	}
	return bandrScripts
}

func (s BackupAndRestoreScripts) BackupOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isBackup() {
			scripts  = append(scripts , script)
		}
	}
	return scripts
}

func (s BackupAndRestoreScripts) RestoreOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isRestore() {
			scripts  = append(scripts , script)
		}
	}
	return scripts
}