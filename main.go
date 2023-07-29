package main

import (
	"log"
	"time"

	"github.com/dkartachov/cube/task"
	"github.com/dkartachov/cube/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

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

	api.Start()
}

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
