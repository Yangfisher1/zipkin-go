// A simple global counter for counting spans
package zipkin

import (
	"sync/atomic"
)

type GlobalCounter int64

var (
	// Make them exported
	GeneratedSpanCounter GlobalCounter
	ReportedSpanCounter  GlobalCounter
)

func (c *GlobalCounter) Set(value int64) {
	atomic.StoreInt64((*int64)(c), value)
}

func (c *GlobalCounter) Inc(value int64) int64 {
	return atomic.AddInt64((*int64)(c), value)
}

func (c *GlobalCounter) Get() int64 {
	return atomic.LoadInt64((*int64)(c))
}
