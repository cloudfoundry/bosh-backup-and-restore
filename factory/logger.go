package factory

import (
	"io"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func BuildLogger(debug bool) boshlog.Logger {
	return BuildBoshLogger(debug)
}

var ApplicationLoggerStdout = writer.NewPausableWriter(os.Stdout)
var ApplicationLoggerStderr = writer.NewPausableWriter(os.Stderr)

func BuildBoshLogger(debug bool) boshlog.Logger {
	if debug {
		return boshlog.NewWriterLogger(boshlog.LevelDebug, ApplicationLoggerStdout)
	}
	return boshlog.NewWriterLogger(boshlog.LevelInfo, ApplicationLoggerStdout)
}

func BuildBoshLoggerWithCustomBuffer(debug bool, buffer io.Writer) boshlog.Logger {
	if debug {
		return boshlog.NewWriterLogger(boshlog.LevelDebug, writer.NewPausableWriter(buffer))
	}
	return boshlog.NewWriterLogger(boshlog.LevelInfo, writer.NewPausableWriter(buffer))
}
