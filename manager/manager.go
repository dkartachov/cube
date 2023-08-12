package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

func (m *Manager) GetTasks() []task.Task {
	tasks := make([]task.Task, 0, len(m.TaskDb))

	for _, t := range m.TaskDb {
		tasks = append(tasks, *t)
	}

	return tasks
}

func (m *Manager) SendWork() {
	if m.Pending.Len() == 0 {
		log.Printf("[manager] no work in queue")

		return
	}

	w := m.SelectWorker()
	teInterface := m.Pending.Dequeue()
	te := teInterface.(task.TaskEvent)
	t := te.Task

	log.Printf("[manager] pulled %v off pending queue", t.ID)

	m.EventDb[te.ID] = &te
	m.TaskWorkerMap[t.ID] = w
	m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], t.ID)

	t.State = task.Scheduled

	m.TaskDb[t.ID] = &t

	data, err := json.Marshal(te)

	if err != nil {
		log.Printf("[manager] unable to marshal task event %v: %v", te, err)

		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

	if err != nil {
		log.Printf("[manager] error connecting to worker %s: %v", w, err)
		m.Pending.Enqueue(t)

		return
	}

	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		// TODO handle error response
		return
	}

	// CHECKME remove this code?
	t = task.Task{}
	err = d.Decode(&t)

	if err != nil {
		log.Printf("[manager] error decoding response body: %v", err)

		return
	}
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

func (m *Manager) UpdateTasks() {
	for {
		log.Printf("[manager] checking for task updates from workers")
		m.updateTasks()
		log.Printf("[manager] sleeping for 15 seconds...")
		time.Sleep(time.Second * 15)
	}
}

func (m *Manager) ProcessTasks() {
	for {
		log.Printf("[manager] processing tasks in the queue")
		m.SendWork()
		log.Printf("[manager] sleeping for 10 seconds...")
		time.Sleep(time.Second * 10)
	}
}

func (m *Manager) updateTasks() {
	// query workers to get list of tasks
	for _, w := range m.Workers {
		log.Printf("[manager] updating tasks for worker %s...", w)

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Get(url)

		if err != nil {
			log.Printf("[manager] error connecting to %s: %v", w, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[manager] error getting tasks from %s: %v", w, err)
			resp.Body.Close()
			continue
		}

		d := json.NewDecoder(resp.Body)
		d.DisallowUnknownFields()

		var tasks []task.Task

		err = d.Decode(&tasks)

		if err != nil {
			log.Printf("[manager] error decoding response body from %s: %v", w, err)
			resp.Body.Close()
			continue
		}

		// update each task's state in manager database to match worker's state
		for _, t := range tasks {
			log.Printf("[manager] updating task %v", t.ID)

			_, ok := m.TaskDb[t.ID]

			if !ok {
				// CHECKME should this be a panic?
				log.Panicf("[manager] task %v not found", t.ID)
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
