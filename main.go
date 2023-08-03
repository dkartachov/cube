package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dkartachov/cube/manager"
	"github.com/dkartachov/cube/task"
	"github.com/dkartachov/cube/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()

			if result.Error != nil {
				log.Printf("error running task: %v", result.Error)
			}
		} else {
			log.Printf("No tasks found")
		}

		log.Printf("Sleeping for 10 seconds")
		time.Sleep(time.Second * 10)
	}
}

func main() {
	address := "localhost"
	port := 1337

	log.Printf("Starting Cube worker at %s:%d", address, port)

	w := worker.Worker{
		Name:  "worker-1",
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.Api{
		Address: address,
		Port:    port,
		Worker:  &w,
	}

	go runTasks(&w)
	go w.CollectStats()
	go api.Start()

	workers := []string{fmt.Sprintf("%s:%d", address, port)}
	m := manager.New(workers)

	// add some random tasks
	for i := 0; i < 3; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("task-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}
		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}

		m.AddTask(te)
		m.SendWork()
	}

	for {
		log.Printf("[Manager] Updating tasks from %d workers...", len(workers))
		m.UpdateTasks()
		time.Sleep(time.Second * 15)
	}
}
