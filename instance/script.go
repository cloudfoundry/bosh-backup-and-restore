package instance

import (
	"path/filepath"
	"strings"
)

type Script string

const (
	backupScriptName            = "backup"
	restoreScriptName           = "restore"
	metadataScriptName          = "metadata"
	preBackupLockScriptName     = "pre-backup-lock"
	preRestoreLockScriptName    = "pre-restore-lock"
	postBackupUnlockScriptName  = "post-backup-unlock"
	postRestoreUnlockScriptName = "post-restore-unlock"

	jobBaseDirectory               = "/var/vcap/jobs/"
	jobDirectoryMatcher            = jobBaseDirectory + "*/bin/bbr/"
	mySQLBackupScriptMatcher       = jobBaseDirectory + "mysql-backup/bin/bbr/*"
	mySQLRestoreScriptMatcher      = jobBaseDirectory + "mysql-restore/bin/bbr/*"
	backupScriptMatcher            = jobDirectoryMatcher + backupScriptName
	restoreScriptMatcher           = jobDirectoryMatcher + restoreScriptName
	metadataScriptMatcher          = jobDirectoryMatcher + metadataScriptName
	preBackupLockScriptMatcher     = jobDirectoryMatcher + preBackupLockScriptName
	preRestoreLockScriptMatcher    = jobDirectoryMatcher + preRestoreLockScriptName
	postBackupUnlockScriptMatcher  = jobDirectoryMatcher + postBackupUnlockScriptName
	postRestoreUnlockScriptMatcher = jobDirectoryMatcher + postRestoreUnlockScriptName
)

func (s Script) isBackup() bool {
	match, _ := filepath.Match(backupScriptMatcher, string(s)) //nolint:errcheck
	return match
}

func (s Script) isRestore() bool {
	match, _ := filepath.Match(restoreScriptMatcher, string(s)) //nolint:errcheck
	return match
}

func (s Script) isMetadata() bool {
	match, _ := filepath.Match(metadataScriptMatcher, string(s)) //nolint:errcheck
	return match
}

func (s Script) isPreBackupUnlock() bool {
	match, _ := filepath.Match(preBackupLockScriptMatcher, string(s)) //nolint:errcheck
	return match
}

func (s Script) isPreRestoreLock() bool {
	match, _ := filepath.Match(preRestoreLockScriptMatcher, string(s)) //nolint:errcheck
	return match
}

func (s Script) isPostBackupUnlock() bool {
	match, _ := filepath.Match(postBackupUnlockScriptMatcher, string(s)) //nolint:errcheck
	return match
}

func (s Script) isPostRestoreUnlock() bool {
	match, _ := filepath.Match(postRestoreUnlockScriptMatcher, string(s)) //nolint:errcheck
	return match
}

func (s Script) isMySQLScript() bool {
	backupMatch, _ := filepath.Match(mySQLBackupScriptMatcher, string(s))   //nolint:errcheck
	restoreMatch, _ := filepath.Match(mySQLRestoreScriptMatcher, string(s)) //nolint:errcheck
	return backupMatch || restoreMatch
}

func (s Script) isPlatformScript() bool {
	if s.isMySQLScript() {
		return false
	}

	return s.isBackup() ||
		s.isRestore() ||
		s.isPreBackupUnlock() ||
		s.isPreRestoreLock() ||
		s.isPostBackupUnlock() ||
		s.isPostRestoreUnlock() ||
		s.isMetadata()
}

func (s Script) splitPath() []string {
	strippedPrefix := strings.TrimPrefix(string(s), jobBaseDirectory)
	splitFirstElement := strings.SplitN(strippedPrefix, "/", 4)
	return splitFirstElement
}

func (s Script) JobName() string {
	pathSplit := s.splitPath()
	return pathSplit[0]
}

func (s Script) Name() string {
	pathSplit := s.splitPath()
	return pathSplit[len(pathSplit)-1]
}
