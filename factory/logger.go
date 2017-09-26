package factory

import (
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func BuildLogger(debug bool) boshlog.Logger {
	return BuildBoshLogger(debug)
}

var ApplicationLoggerStdout *writer.PausableWriter = writer.NewPausableWriter(os.Stdout)
var ApplicationLoggerStderr *writer.PausableWriter = writer.NewPausableWriter(os.Stderr)

func BuildBoshLogger(debug bool) boshlog.Logger {
	if debug {
		return boshlog.NewWriterLogger(boshlog.LevelDebug, ApplicationLoggerStdout, ApplicationLoggerStderr)
	}
	return boshlog.NewWriterLogger(boshlog.LevelInfo, ApplicationLoggerStdout, ApplicationLoggerStderr)
}
