package irods

import (
	"sync"

	"k8s.io/klog"
)

// ParallelJobTask is a task function for a parallel job
type ParallelJobTask func(job *ParallelJob) error

// ParallelJob is a parallel job
type ParallelJob struct {
	manager *ParallelJobManager

	index             int64
	name              string
	task              ParallelJobTask
	threadsRequired   int
	barrierBeforeTask bool
	lastError         error
}

// GetManager returns manager
func (job *ParallelJob) GetManager() *ParallelJobManager {
	return job.manager
}

func newParallelJob(manager *ParallelJobManager, index int64, name string, task ParallelJobTask, threadsRequired int, barrierBeforeTask bool) *ParallelJob {
	return &ParallelJob{
		manager:           manager,
		index:             index,
		name:              name,
		task:              task,
		threadsRequired:   threadsRequired,
		barrierBeforeTask: barrierBeforeTask,
		lastError:         nil,
	}
}

// ParallelJobManager manages parallel jobs
type ParallelJobManager struct {
	nextJobIndex int64
	pendingJobs  chan *ParallelJob
	maxThreads   int
	lastError    error
	mutex        sync.RWMutex

	availableThreadWaitCondition *sync.Cond // used for checking available threads
	scheduleWait                 sync.WaitGroup
	jobWait                      sync.WaitGroup
}

// NewParallelJobManager creates a new ParallelJobManager
func NewParallelJobManager(maxThreads int) *ParallelJobManager {
	manager := &ParallelJobManager{
		nextJobIndex: 0,
		pendingJobs:  make(chan *ParallelJob, 100),
		maxThreads:   maxThreads,
		lastError:    nil,
		mutex:        sync.RWMutex{},

		scheduleWait: sync.WaitGroup{},
		jobWait:      sync.WaitGroup{},
	}

	manager.availableThreadWaitCondition = sync.NewCond(&manager.mutex)

	manager.scheduleWait.Add(1)

	return manager
}

func (manager *ParallelJobManager) getNextJobIndex() int64 {
	idx := manager.nextJobIndex
	manager.nextJobIndex++
	return idx
}

// Schedule schedules a new task
func (manager *ParallelJobManager) Schedule(name string, task ParallelJobTask, threadsRequired int) error {
	manager.mutex.Lock()

	// do not accept new schedule if there's an error
	if manager.lastError != nil {
		defer manager.mutex.Unlock()
		return manager.lastError
	}

	job := newParallelJob(manager, manager.getNextJobIndex(), name, task, threadsRequired, false)

	// release lock since adding to chan may block
	manager.mutex.Unlock()

	manager.pendingJobs <- job
	manager.jobWait.Add(1)

	return nil
}

// Schedule schedules a new task
func (manager *ParallelJobManager) ScheduleBarrier(name string) error {
	manager.mutex.Lock()

	// do not accept new schedule if there's an error
	if manager.lastError != nil {
		defer manager.mutex.Unlock()
		return manager.lastError
	}

	barrierTask := func(job *ParallelJob) error {
		// do nothing
		return nil
	}

	job := newParallelJob(manager, manager.getNextJobIndex(), name, barrierTask, 1, true)

	// release lock since adding to chan may block
	manager.mutex.Unlock()

	manager.pendingJobs <- job
	manager.jobWait.Add(1)

	return nil
}

// DoneScheduling completes scheduling
func (manager *ParallelJobManager) DoneScheduling() {
	close(manager.pendingJobs)
	manager.scheduleWait.Done()
}

// Wait waits for pending tasks
func (manager *ParallelJobManager) Wait() error {
	klog.V(5).Info("waiting schedule-wait")
	manager.scheduleWait.Wait()
	klog.V(5).Info("waiting job-wait")
	manager.jobWait.Wait()

	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.lastError
}

// Start starts processing tasks
func (manager *ParallelJobManager) Start() {
	go func() {
		klog.V(5).Info("start job run thread")
		defer klog.V(5).Info("exit job run thread")

		currentThreads := 0

		for job := range manager.pendingJobs {
			cont := true

			manager.mutex.RLock()
			if manager.lastError != nil {
				cont = false
			}
			manager.mutex.RUnlock()

			if cont {
				manager.mutex.Lock()
				if currentThreads > 0 {
					for currentThreads+job.threadsRequired > manager.maxThreads {
						// exceeds max threads
						// wait until it becomes available
						klog.V(5).Infof("waiting for other jobs to complete - current %d, max %d", currentThreads, manager.maxThreads)

						manager.availableThreadWaitCondition.Wait()
					}

					if job.barrierBeforeTask {
						for currentThreads > 0 {
							// this job requires no current jobs running
							// wait until all jobs are done
							klog.V(5).Infof("waiting for other jobs to complete - current %d", currentThreads)
							manager.availableThreadWaitCondition.Wait()
						}
					}
				}

				currentThreads += job.threadsRequired

				go func(pjob *ParallelJob) {
					klog.V(5).Infof("Run job %d, %s", pjob.index, pjob.name)

					err := pjob.task(pjob)

					if err != nil {
						// mark error
						manager.mutex.Lock()
						manager.lastError = err
						manager.mutex.Unlock()

						klog.Error(err)
						// don't stop here
					}

					currentThreads -= pjob.threadsRequired

					manager.jobWait.Done()

					manager.mutex.Lock()
					manager.availableThreadWaitCondition.Broadcast()
					manager.mutex.Unlock()
				}(job)

				manager.mutex.Unlock()
			} else {
				manager.jobWait.Done()
			}
		}
		manager.jobWait.Wait()
	}()
}
