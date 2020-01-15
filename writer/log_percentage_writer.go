package writer

import (
	"io"
)

type LogPercentageWriter struct {
	Writer              io.Writer
	bytesWritten        int
	logger              Logger
	totalSize           int
	command             string
	message             string
	lastLogPercentage   int
	percentageIncrement int
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Info(tag, msg string, args ...interface{})
}

func NewLogPercentageWriter(writer io.Writer, logger Logger, totalSize int, command, message string) *LogPercentageWriter {
	return &LogPercentageWriter{
		Writer:              writer,
		logger:              logger,
		totalSize:           totalSize,
		command:             command,
		message:             message,
		percentageIncrement: 5,
	}
}

func (l *LogPercentageWriter) Write(b []byte) (int, error) {
	n, err := l.Writer.Write(b)
	if err != nil {
		return 0, err
	}

	l.bytesWritten += n
	percentageWrittenSoFar := (100 * l.bytesWritten) / l.totalSize

	if l.bytesWritten > l.totalSize {
		l.logger.Info(l.command, l.message, 100)
	} else if percentageWrittenSoFar >= l.lastLogPercentage+l.percentageIncrement {
		l.logger.Info(l.command, l.message, percentageWrittenSoFar)
	}

	return n, nil
}
