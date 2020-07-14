package queue

import (
	"context"
	"sync"

	. "github.com/progfay/go-job-queue/job"
)

type Queue struct {
	jobs   chan *Job
	queue  []*Job
	mu     *sync.Mutex
	wg     *sync.WaitGroup
	ctx    context.Context
	cancel func()
}

func NewQueue(maxJobCount int) *Queue {
	ctx, cancel := context.WithCancel(context.Background())

	q := &Queue{
		jobs:   make(chan *Job, maxJobCount),
		queue:  []*Job{},
		mu:     &sync.Mutex{},
		wg:     &sync.WaitGroup{},
		ctx:    ctx,
		cancel: cancel,
	}
	defer func() {
		go q.start()
	}()

	return q
}

func (q *Queue) start() {
	for {
		select {
		case <-q.ctx.Done():
			q.wg.Done()
			return

		case j := <-q.jobs:
			func() {
				go func(job *Job) {
					defer q.wg.Done()
					job.Handler(job.Args...)

					q.mu.Lock()
					defer q.mu.Unlock()
					if len(q.queue) == 0 {
						return
					}

					j := q.queue[0]
					q.queue = q.queue[1:]
					q.jobs <- j
				}(j)
			}()
		}
	}
}

func (q *Queue) Add(job *Job) {
	if q.ctx.Err() != nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.wg.Add(1)
	if len(q.jobs) < cap(q.jobs) {
		q.jobs <- job
	} else {
		q.queue = append(q.queue, job)
	}
}

func (q *Queue) Wait() {
	q.wg.Wait()
}

func (q *Queue) Stop() {
	q.cancel()
}
