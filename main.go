package main

import (
	"fmt"
	"log"

	"github.com/dkartachov/cube/manager"
	"github.com/dkartachov/cube/task"
	"github.com/dkartachov/cube/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	mHost := "localhost"
	mPort := 1337
	wHost := "localhost"
	wPort := 1338

	w := worker.Worker{
		Name:  "worker-1",
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	wApi := worker.Api{
		Address: wHost,
		Port:    wPort,
		Worker:  &w,
	}

	log.Printf("starting Cube worker at %s:%d", wHost, wPort)

	go w.RunTasks()
	go w.CollectStats()
	go wApi.Start()

	workers := []string{fmt.Sprintf("%s:%d", wHost, wPort)}
	m := manager.New(workers)
	mApi := manager.Api{
		Address: mHost,
		Port:    mPort,
		Manager: m,
	}

	log.Printf("starting Cube manager at %s:%d", mHost, mPort)

	go m.ProcessTasks()
	go m.UpdateTasks()

	mApi.Start()
}
