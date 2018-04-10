package main

// lifted from https://github.com/alexsasharegan/go-simple-tcp-server/blob/master/counter.go

import (
	"log"
	"sync"
	"time")

// Counter is a container for tracking and managing the runtime counters.
type Counter struct {
	mu sync.RWMutex
	// Uniq is a map of the unique numbers received during uptime.
	Uniq map[int]bool
	// Cnt valid numbers received during uptime.
	Cnt int
	// IntvlCnt is the total valid numbers received during output interval.
	IntvlCnt int

	intvl *struct {
		output  chan bool
		logging chan bool
	}
	// Sem is a semaphore to do request limiting.
	Sem chan int
}


// NewCounter constructs a new Counter.
func NewCounter(connLimit int) *Counter {
 	return &Counter{
		Uniq: make(map[int]bool),
		Sem:  make(chan int, connLimit),

		intvl: &struct {
			output  chan bool
			logging chan bool
		}{
			output:  make(chan bool),
			logging: make(chan bool),
		},
	}
}



// RecordUniq adds a unique int to the map and the log buffer in a thread safe way.
func (c *Counter) RecordUniq(num int) (err error) {
	c.mu.Lock()
	c.Uniq[num] = true
 	c.mu.Unlock()
	return err
}

func (c *Counter) outputCounters() {
	// We could use a read lock first,
	// then grab a write lock to clear counter.
	c.mu.Lock()

	log.Printf(
		"----------------\n"+
			"Count unique: %d\n"+
			"Count total : %d\n"+
			"Count last  : %d\n",
		len(c.Uniq),
		c.Cnt,
		c.IntvlCnt)
	c.IntvlCnt = 0

	c.mu.Unlock()
}

// RunOutputInterval outputs the counters on an interval.
// It takes a nil channel that the caller will close to stop execution.
// Must be run on go routine.
func (c *Counter) RunOutputInterval(intvl time.Duration) {
	for {
		select {
		case <-time.After(intvl):
			c.outputCounters()
		case <-c.intvl.output:
			return
		}
	}
}

// StopOutputIntvl exits the output interval by closing it's underlying nil channel.
func (c *Counter) StopOutputIntvl() {
	close(c.intvl.output)
}

// RunLogInterval outputs the counters on an interval.
// It takes a nil channel that the caller will close to stop execution.
// Must be run on go routine.
func (c *Counter) RunLogInterval(intvl time.Duration) {
	var err error
	for {
		select {
		case <-time.After(intvl):
 			if err != nil {
				log.Fatalf("could not flush and rotate logs: %v", err)
			}
		case <-c.intvl.logging:
			if err != nil {
				log.Printf( "error flushing log to disk: %v", err)
			}
			return
		}
	}
}


// Inc increments the counters in a thread safe way.
func (c *Counter) Inc() {
	c.mu.Lock()
	c.Cnt++
	c.IntvlCnt++
	c.mu.Unlock()
}

// HasValue checks if an int has been recorded in a thread safe way.
func (c *Counter) HasValue(num int) (b bool) {
	c.mu.RLock()
	b = c.Uniq[num]
	c.mu.RUnlock()
	return
}

// Close closes all internals and flushes logs to disk.
func (c *Counter) Close() (err error) {
	c.StopOutputIntvl()
	return
}