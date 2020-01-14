package writer

import (
	"io"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

type LogPercentageWriter struct {
	Writer       io.Writer
	bytesWritten int
	logger       orchestrator.Logger
	totalSize    int
	command      string
	message      string
}

func NewLogPercentageWriter(writer io.Writer, logger orchestrator.Logger, totalSize int, command, message string) *LogPercentageWriter {
	return &LogPercentageWriter{
		Writer:    writer,
		logger:    logger,
		totalSize: totalSize,
		command:   command,
		message:   message,
	}
}

func (l *LogPercentageWriter) Write(b []byte) (int, error) {
	n, err := l.Writer.Write(b)
	if err != nil {
		return 0, err
	}

	l.bytesWritten += n

	if l.bytesWritten > l.totalSize {
		l.logger.Info(l.command, l.message, 100)
	} else {
		l.logger.Info(l.command, l.message, ((100 * l.bytesWritten) / l.totalSize))
	}

	return n, nil
}
