package common

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type QueueOptions struct {
	Workers     int // Number of workers to process jobs. (i.e. goroutines)
	ChannelSize int // Size of the channel to hold jobs. (i.e. buffered channel)
}

// Creates a new job queue with the given options.
// If the number of workers is not specified, it defaults to 1.
// If the channel size is not specified, it defaults to 100.
func NewQueue(options QueueOptions) *Queue {
	if options.Workers == 0 {
		options.Workers = 1
	}
	if options.ChannelSize == 0 {
		options.ChannelSize = 100
	}

	return &Queue{
		IsRunning: 0,
		Workers:   options.Workers,
		Channel:   make(chan Job, options.ChannelSize),
		Lock: Lock{
			jobs: make(map[string]struct {
				running bool
				lastRun time.Time
			}),
		},
	}
}

type Queue struct {
	IsRunning int32    // Flag to indicate if the queue is running.
	Workers   int      // Number of workers to process jobs. (i.e. goroutines)
	Channel   chan Job // Channel to hold jobs. (i.e. buffered channel)
	Lock      Lock     // Job lock manager. (i.e. prevents concurrent runs if job is lockable)
}

// Starts processing the jobs in the queue.
func (q *Queue) StartJobQueue() {
	atomic.StoreInt32(&q.IsRunning, 1) // Set the queue as running.
	for i := 0; i < q.Workers; i++ {
		// Start a goroutine for each worker.
		go func() {
			// Loop indefinitely to process jobs.
			for job := range q.Channel {
				if atomic.LoadInt32(&q.IsRunning) == 0 {
					return // Exit goroutine if the queue is not running.
				}

				var err error
				// If the job is lockable, lock it to prevent concurrent runs.
				if job.Lockable {
					_, err = q.Lock.Lock(job.Name)
					if err != nil { // Skip the job if it's already running.
						fmt.Printf("failed to lock job %s: %v\n", job.Name, err)
						continue
					}
					// Execute the job and unlock it when done.
					err = job.Func()
					q.Lock.Unlock(job.Name)
				} else { // Execute the job if it's not lockable.
					err = job.Func()
				}
				if err != nil {
					fmt.Printf("failed to execute job %s: %v\n", job.Name, err)
				}
			}
		}()
	}
}

// Stops processing the jobs in the queue, waits for all jobs to finish processing.
func (q *Queue) StopJobQueue() {
	atomic.StoreInt32(&q.IsRunning, 0) // Set the queue as not running to prevent new jobs.
	close(q.Channel)                   // Close the job queue channel.
	for len(q.Channel) > 0 {
		time.Sleep(100 * time.Millisecond) // Wait for jobs to finish processing.
	}
}

// Attempts to add a job to the queue. Fails if the queue is not running or if the queue is full.
func (q *Queue) AddJob(job Job) error {
	if atomic.LoadInt32(&q.IsRunning) == 0 { // Check if the queue is running.
		return fmt.Errorf("job queue is not running")
	}
	select {
	case q.Channel <- job: // Add the job to the queue if there's space.
		return nil
	default: // Fails if the queue is full.
		return fmt.Errorf("job queue is full")
	}
}

// Describes a job type with a name, function and lockable flag.
type Job struct {
	Name     string       // Unique name for the job (you can use params into the name if needed).
	Func     func() error // Function to execute the job.
	Lockable bool         // If true the job (exact same name) can't be run concurrently.
}

// Manages job execution states to prevent concurrent runs.
type Lock struct {
	mu   sync.Mutex
	jobs map[string]struct {
		running bool
		lastRun time.Time
	}
}

// Attempts to lock a job for execution.
func (l *Lock) Lock(jobName string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	job, ok := l.jobs[jobName]
	if ok && job.running {
		return false, fmt.Errorf("job %s is already running", jobName)
	}
	// Update the job's state whether it's new or existing.
	l.jobs[jobName] = struct {
		running bool
		lastRun time.Time
	}{running: true, lastRun: time.Now()}
	return true, nil
}

// Releases the lock on a job.
func (l *Lock) Unlock(jobName string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.jobs, jobName)
}
