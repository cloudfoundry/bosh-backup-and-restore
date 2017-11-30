package orchestrator

type BackupChecksum map[string]string

func (b BackupChecksum) Match(other BackupChecksum) (bool, []string) {
	if len(b) != len(other) {
		return false, b.getMismatchedFiles(other)
	}

	for key := range b {
		if b[key] != other[key] {
			return false, b.getMismatchedFiles(other)
		}
	}

	return true, []string{}
}

func (b BackupChecksum) getMismatchedFiles(other BackupChecksum) []string {
	var files []string

	for key := range b {
		if b[key] != other[key] {
			files = append(files, key)
		}
	}

	return files
}