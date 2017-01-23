package bosh

import (
	"path/filepath"
	"strings"
	"fmt"
)

type BackupAndRestoreScripts []Script

const (
	backupScriptName  = "p-backup"
	restoreScriptName = "p-restore"
	preBackupScriptName = "p-pre-backup-lock"

	jobBaseDirectory = "/var/vcap/jobs/"
	jobDirectoryMatcher  = jobBaseDirectory + "*/bin/"
	backupScriptMatcher  = jobDirectoryMatcher + backupScriptName
	restoreScriptMatcher = jobDirectoryMatcher + restoreScriptName
	preBackupUnlockScriptMatcher = jobDirectoryMatcher + preBackupScriptName
)

type Script string

func (s Script) isBackup() bool {
	match, _ := filepath.Match(backupScriptMatcher, string(s))
	return match
}

func (s Script) isRestore() bool {
	match, _ := filepath.Match(restoreScriptMatcher, string(s))
	return match
}

func (s Script) isPreBackupUnlock() bool {
	match, _ := filepath.Match(preBackupUnlockScriptMatcher, string(s))
	return match
}

func (s Script) isPlatformScript() bool {
	return s.isBackup() || s.isRestore() || s.isPreBackupUnlock()
}

func (s Script) JobName() (string, error) {
	if !strings.HasPrefix(string(s), jobBaseDirectory) {
		return "", fmt.Errorf("script %s is not a job script", string(s))
	}

	strippedPrefix := strings.TrimPrefix(string(s), jobBaseDirectory)
	splitFirstElement := strings.SplitN(strippedPrefix, "/", 2)
	return splitFirstElement[0], nil
}

func NewBackupAndRestoreScripts(files []string) BackupAndRestoreScripts {
	bandrScripts := []Script{}
	for _, s := range files {
		s:=Script(s)
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

func (s BackupAndRestoreScripts) PreBackupLockOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isPreBackupUnlock() {
			scripts  = append(scripts , script)
		}
	}
	return scripts
}