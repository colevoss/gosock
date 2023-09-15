package gosock

import (
	"log"
	"time"
)

type PoolTask func()
type PoolEventHandler func(id int)

type Pool struct {
	jobs          chan PoolTask
	sem           chan struct{}
	maxPools      int
	ttl           time.Duration
	id            int
	eventHandlers map[string]PoolEventHandler
}

func NewPool(queue, maxPools int, ttl time.Duration) *Pool {
	return &Pool{
		id: 0,
		// Amount of jobs that can be queued once maxPools is full
		jobs: make(chan PoolTask, queue),
		// Allows maxPools number of goroutines to spawn
		sem:           make(chan struct{}, maxPools),
		ttl:           ttl,
		eventHandlers: make(map[string]PoolEventHandler),
	}
}

func (p *Pool) release() {
	<-p.sem
}

func (p *Pool) Handle(handlerInit PoolHandlerInit) *Pool {
	handlerInit(p)
	return p
}

type PoolHandlerInit func(*Pool)

func PoolOpen(handler PoolEventHandler) PoolHandlerInit {
	return func(pool *Pool) {
		pool.eventHandlers["open"] = handler
	}
}

func PoolClose(handler PoolEventHandler) PoolHandlerInit {
	return func(pool *Pool) {
		pool.eventHandlers["close"] = handler
	}
}

func (p *Pool) close(id int) {
	p.release()
	if handler, ok := p.eventHandlers["close"]; ok {
		handler(id)
	}
}

func (p *Pool) open(id int) {
	if handler, ok := p.eventHandlers["open"]; ok {
		handler(id)
	}
}

func (p *Pool) Schedule(task PoolTask) {
	log.Printf("Scheduling task")
	select {
	// If we can aquire the semaphore
	case p.sem <- struct{}{}:
		// Spawn new go routine
		id := p.id
		go p.workTimeout(task, id)
		p.id++
	default:
		// Otherwise just queue the task
		p.jobs <- task
	}

}

func (p *Pool) workTimeout(task PoolTask, id int) {
	p.open(id)
	defer p.close(id)

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
