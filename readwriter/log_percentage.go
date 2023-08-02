package readwriter

import (
	"io"
)

type LogPercentage struct {
	logger              Logger
	totalSize           int
	command             string
	message             string
	lastLogPercentage   int
	percentageIncrement int
}

type LogPercentageWriter struct {
	Writer       io.Writer
	bytesWritten int
	LogPercentage
}

type LogPercentageReader struct {
	Reader    io.Reader
	bytesRead int
	LogPercentage
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Info(tag, msg string, args ...interface{})
}

func NewLogPercentageWriter(writer io.Writer, logger Logger, totalSize int, command, message string) *LogPercentageWriter {
	return &LogPercentageWriter{
		Writer: writer,
		LogPercentage: LogPercentage{
			logger:              logger,
			totalSize:           totalSize,
			command:             command,
			message:             message,
			percentageIncrement: 5,
		},
	}
}

func (lw *LogPercentageWriter) Write(b []byte) (int, error) {
	n, err := lw.Writer.Write(b)
	if err != nil {
		return 0, err
	}

	lw.bytesWritten += n
	lw.logPercentage(n, lw.bytesWritten)

	return n, nil
}

func NewLogPercentageReader(reader io.Reader, logger Logger, totalSize int, command, message string) *LogPercentageReader {
	return &LogPercentageReader{
		Reader: reader,
		LogPercentage: LogPercentage{
			logger:              logger,
			totalSize:           totalSize,
			command:             command,
			message:             message,
			percentageIncrement: 5,
		},
	}
}

func (lr *LogPercentageReader) Read(b []byte) (int, error) {
	n, err := lr.Reader.Read(b)
	if err != nil {
		return 0, err
	}

	lr.bytesRead += n
	lr.logPercentage(n, lr.bytesRead)

	return n, nil
}

func (l *LogPercentage) logPercentage(n, b int) {
	percentageWrittenSoFar := (100 * b) / l.totalSize
	if b > l.totalSize {
		l.logger.Info(l.command, l.message, 100)
	} else if percentageWrittenSoFar >= l.lastLogPercentage+l.percentageIncrement {
		l.lastLogPercentage = percentageWrittenSoFar
		l.logger.Info(l.command, l.message, percentageWrittenSoFar)
	}
}
