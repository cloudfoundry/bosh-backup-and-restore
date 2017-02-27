package instance

import (
	"path/filepath"
	"strings"
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

func (s Script) isMetadata() bool {
	match, _ := filepath.Match(metadataScriptMatcher, string(s))
	return match
}

func (s Script) isPreBackupUnlock() bool {
	match, _ := filepath.Match(preBackupLockScriptMatcher, string(s))
	return match
}

func (s Script) isPostBackupUnlock() bool {
	match, _ := filepath.Match(postBackupUnlockScriptMatcher, string(s))
	return match
}

func (s Script) isPlatformScript() bool {
	return s.isBackup() ||
		s.isRestore() ||
		s.isPreBackupUnlock() ||
		s.isPostBackupUnlock() ||
		s.isMetadata()
}

func (s Script) splitPath() []string {
	strippedPrefix := strings.TrimPrefix(string(s), jobBaseDirectory)
	splitFirstElement := strings.SplitN(strippedPrefix, "/", 3)
	return splitFirstElement
}

func (s Script) JobName() string {
	pathSplit := s.splitPath()
	return pathSplit[0]
}

func (script Script) Name() string {
	pathSplit := script.splitPath()
	return pathSplit[len(pathSplit)-1]
}
