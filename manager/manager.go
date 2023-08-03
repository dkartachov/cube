package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/dkartachov/cube/task"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue // holds TaskEvent
	TaskDb        map[uuid.UUID]*task.Task
	EventDb       map[uuid.UUID]*task.TaskEvent
	Workers       []string // this array will hold worker addresses in the form <hostname>:<port>
	WorkerTaskMap map[string][]uuid.UUID
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

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) UpdateTasks() {
	// query workers to get list of tasks
	for _, w := range m.Workers {
		log.Printf("updating tasks for worker %s...", w)

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Get(url)

		if err != nil {
			log.Printf("error connecting to %s: %v", w, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("error getting tasks from %s: %v", w, err)
			resp.Body.Close()
			continue
		}

		d := json.NewDecoder(resp.Body)
		d.DisallowUnknownFields()

		var tasks []task.Task

		err = d.Decode(&tasks)

		if err != nil {
			log.Printf("error decoding response body from %s: %v", w, err)
			resp.Body.Close()
			continue
		}

		// update each task's state in manager database to match worker's state
		for _, t := range tasks {
			log.Printf("updating task %v", t.ID)

			_, ok := m.TaskDb[t.ID]

			if !ok {
				// CHECKME should this be fatal?
				log.Fatalf("task %v not found", t.ID)
			}

			if m.TaskDb[t.ID].State != t.State {
				m.TaskDb[t.ID].State = t.State
			}

			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerId = t.ContainerId
		}
	}
}

func (m *Manager) SendWork() {
	if m.Pending.Len() == 0 {
		log.Printf("no work in the queue")

		return
	}

	w := m.SelectWorker()
	teInterface := m.Pending.Dequeue()
	te := teInterface.(task.TaskEvent)
	t := te.Task

	log.Printf("pulled %v off pending queue", t)

	m.EventDb[te.ID] = &te
	m.TaskWorkerMap[t.ID] = w
	m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], t.ID)

	t.State = task.Scheduled

	m.TaskDb[t.ID] = &t

	data, err := json.Marshal(te)

	if err != nil {
		log.Printf("unable to marshal task event %v: %v", te, err)

		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

	if err != nil {
		log.Printf("error connecting to worker %s: %v", w, err)
		m.Pending.Enqueue(t)

		return
	}

	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		// TODO handle error response
		return
	}

	t = task.Task{}
	err = d.Decode(&t)

	if err != nil {
		log.Printf("error decoding response body: %v", err)

		return
	}

	log.Printf("%#v", t)
}

func New(workers []string) *Manager {
	workerTaskMap := make(map[string][]uuid.UUID)

	for _, w := range workers {
		workerTaskMap[w] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
		TaskDb:        make(map[uuid.UUID]*task.Task),
		EventDb:       make(map[uuid.UUID]*task.TaskEvent),
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: make(map[uuid.UUID]string),
	}
}
