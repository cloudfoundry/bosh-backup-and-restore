package writer

import (
	"io"
	"sync"
)

type LogPercentageWriter struct {
	Writer  io.Writer
	mutex   sync.RWMutex
	counter int
}

func NewLogPercentageWriter(w io.Writer) *LogPercentageWriter {
	return &LogPercentageWriter{Writer: w}
}

func (c *LogPercentageWriter) Write(b []byte) (int, error) {
	n, err := c.Writer.Write(b)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err != nil {
		return 0, err
	}

	c.counter += n
	return n, nil
}

func (c *LogPercentageWriter) Count() int {
	var localCounter int
	c.mutex.Lock()
	localCounter = c.counter
	c.mutex.Unlock()
	return localCounter
}
