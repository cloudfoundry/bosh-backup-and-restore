package instance

type BackupAndRestoreScripts []Script

const (
	backupScriptName           = "backup"
	restoreScriptName          = "restore"
	metadataScriptName         = "metadata"
	preBackupLockScriptName    = "pre-backup-lock"
	postBackupUnlockScriptName = "post-backup-unlock"

	jobBaseDirectory              = "/var/vcap/jobs/"
	jobDirectoryMatcher           = jobBaseDirectory + "*/bin/bbr/"
	backupScriptMatcher           = jobDirectoryMatcher + backupScriptName
	restoreScriptMatcher          = jobDirectoryMatcher + restoreScriptName
	metadataScriptMatcher         = jobDirectoryMatcher + metadataScriptName
	preBackupLockScriptMatcher    = jobDirectoryMatcher + preBackupLockScriptName
	postBackupUnlockScriptMatcher = jobDirectoryMatcher + postBackupUnlockScriptName
)

func NewBackupAndRestoreScripts(files []string) BackupAndRestoreScripts {
	bandrScripts := []Script{}
	for _, s := range files {
		s := Script(s)
		if s.isPlatformScript() {
			bandrScripts = append(bandrScripts, s)
		}
	}
	return bandrScripts
}
func (s BackupAndRestoreScripts) firstOrBlank() Script {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func (s BackupAndRestoreScripts) HasBackup() bool {
	return len(s.BackupOnly()) > 0
}

func (s BackupAndRestoreScripts) BackupOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isBackup() {
			scripts = append(scripts, script)
		}
	}
	return scripts
}

func (s BackupAndRestoreScripts) MetadataOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isMetadata() {
			scripts = append(scripts, script)
		}
	}
	return scripts
}

func (s BackupAndRestoreScripts) RestoreOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isRestore() {
			scripts = append(scripts, script)
		}
	}
	return scripts
}

func (s BackupAndRestoreScripts) PreBackupLockOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isPreBackupUnlock() {
			scripts = append(scripts, script)
		}
	}
	return scripts
}

func (s BackupAndRestoreScripts) PostBackupUnlockOnly() BackupAndRestoreScripts {
	scripts := BackupAndRestoreScripts{}
	for _, script := range s {
		if script.isPostBackupUnlock() {
			scripts = append(scripts, script)
		}
	}
	return scripts
}
