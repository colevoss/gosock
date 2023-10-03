package gosock

import (
	"log"
	"sync/atomic"
	"time"
)

type PoolTask func()

type PanicHandler func(interface{})

type Pool struct {
	jobs     chan PoolTask
	sem      chan struct{}
	maxPools int
	ttl      time.Duration

	workerCount int32

	panicHandler PanicHandler
}

func NewPool(queue, maxPools int, ttl time.Duration) *Pool {
	return &Pool{
		// This probably needs to be locked or atomic in some way
		workerCount: 0,
		// Amount of jobs that can be queued once maxPools is full
		jobs: make(chan PoolTask, queue),
		// Allows maxPools number of goroutines to spawn
		sem: make(chan struct{}, maxPools),
		ttl: ttl,
	}
}

func (p *Pool) WorkerCount() int32 {
	count := p.workerCount

	return count
}

func (p *Pool) OnPanic(handler PanicHandler) {
	p.panicHandler = handler
}

func (p *Pool) Schedule(task PoolTask) {
	select {
	// If we can aquire the semaphore
	case p.sem <- struct{}{}:
		// Spawn new go routine
		go p.workTimeout(task)
	default:
		// Otherwise just queue the task
		p.jobs <- task
	}
}

func (p *Pool) release() {
	<-p.sem
}

type PoolHandlerInit func(*Pool)

func (p *Pool) close() {
	id := p.workerCount
	atomic.AddInt32(&p.workerCount, -1)
	log.Printf("Closing worker %d", id)
	p.release()
}

func (p *Pool) workTimeout(task PoolTask) {
	atomic.AddInt32(&p.workerCount, 1)

	defer p.close()
	defer func() {
		if p.panicHandler == nil {
			return
		}

		if r := recover(); r != nil {
			p.panicHandler(r)
		}
	}()

	log.Printf("Opening worker %d", p.workerCount)

	task()

	t := time.NewTimer(p.ttl)

	for {
		select {
		// If the timer has elapsed, we should close this
		// worker since it hasn't had work in a while
		case <-t.C:
			return

		case job := <-p.jobs:
			t.Stop()
			job()
			t.Reset(p.ttl)
		}
	}
}
