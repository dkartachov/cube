package manager

import (
	"fmt"

	"github.com/dkartachov/cube/task"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue
	TaskDb        map[string][]task.Task
	EventDb       map[string][]task.TaskEvent
	Workers       []string // this array will hold worker addresses in the form <hostname>:<port>
	WorkerTaskMap map[string]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
}

// "naive" round-robin approach (first pass)
func (m *Manager) SelectWorker() string {
	if m.LastWorker == len(m.Workers)-1 {
		m.LastWorker = 0
		return m.Workers[m.LastWorker]
	}

	m.LastWorker += 1

	return m.Workers[m.LastWorker]
}

func (m *Manager) UpdateTasks() {
	// call to worker.CollectStats will be made eventually
	fmt.Println("Updating tasks")
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()
	}
}
