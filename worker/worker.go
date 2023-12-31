package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dkartachov/cube/task"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *Stats
}

func (w *Worker) CollectStats() {
	for {
		log.Printf("[worker: %s] collecting stats", w.Name)

		w.Stats = GetStats()

		// TODO increment stats task count from worker task count

		time.Sleep(time.Second * 15)
	}
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()

	if t == nil {
		log.Printf("[worker: %s] no tasks found in queue", w.Name)

		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)
	taskPersisted := w.Db[taskQueued.ID]

	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = &taskQueued
	}

	var result task.DockerResult

	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("we should not get here")
		}
	} else {
		err := fmt.Errorf("[worker: %s] invalid transition from %v to %v", w.Name, taskPersisted.State, taskQueued.State)

		result.Error = err
	}

	return result
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()

	c := t.NewConfig()
	d := t.NewDocker(c)

	result := d.Run()

	if result.Error != nil {
		log.Printf("[worker: %s] error running task %s: %v", w.Name, t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t

		return result
	}

	t.ContainerId = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	c := t.NewConfig()
	d := t.NewDocker(c)

	d.ContainerId = t.ContainerId

	result := d.Stop()

	if result.Error != nil {
		log.Printf("[worker: %s] error stopping container %s: %v", w.Name, d.ContainerId, result.Error)

		return result
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db[t.ID] = &t

	log.Printf("[worker: %s] stopped and removed container %s for task %s", w.Name, d.ContainerId, t.ID)

	return result
}

func (w *Worker) GetTasks() []task.Task {
	tasks := make([]task.Task, 0, len(w.Db))

	for _, task := range w.Db {
		tasks = append(tasks, *task)
	}

	return tasks
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()

			if result.Error != nil {
				log.Printf("[worker: %s] error running task: %v", w.Name, result.Error)
			}
		} else {
			log.Printf("[worker: %s] no tasks found", w.Name)
		}

		log.Printf("[worker: %s] sleeping for 10 seconds", w.Name)
		time.Sleep(time.Second * 10)
	}
}
