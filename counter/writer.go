package counter

import (
	"io"
	"sync"
)

type CountWriter struct {
	Writer  io.Writer
	mutex   sync.RWMutex
	counter int
}

func NewCountWriter(w io.Writer) *CountWriter {
	return &CountWriter{Writer: w}
}

func (c *CountWriter) Write(b []byte) (int, error) {
	n, err := c.Writer.Write(b)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err != nil {
		return 0, err
	}

	c.counter += n
	return n, nil
}

func (c *CountWriter) Count() int {
	var localCounter int
	c.mutex.Lock()
	localCounter = c.counter
	c.mutex.Unlock()
	return localCounter
}
