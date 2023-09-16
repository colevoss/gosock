package gosock

import (
	"time"
)

type PoolTask func()

type Pool struct {
	jobs        chan PoolTask
	sem         chan struct{}
	maxPools    int
	ttl         time.Duration
	workerCount int
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

func (p *Pool) WorkerCount() int {
	count := p.workerCount

	return count
}

func (p *Pool) Schedule(task PoolTask) {
	select {
	// If we can aquire the semaphore
	case p.sem <- struct{}{}:
		// Spawn new go routine
		go p.workTimeout(task)
		p.workerCount++
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
	p.workerCount--
	p.release()
}

func (p *Pool) workTimeout(task PoolTask) {
	defer p.close()

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
