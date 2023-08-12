package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dkartachov/cube/task"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Manager.GetTasks())
}

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	var te task.TaskEvent

	if err := d.Decode(&te); err != nil {
		msg := "[manager-api] error decoding json"

		log.Printf("%s: %v", msg, err)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	a.Manager.AddTask(te)

	log.Printf("[manager-api] added task %v", te.Task.ID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	if taskID == "" {
		msg := "[manager-api] missing taskID in request"

		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	taskUUID, err := uuid.Parse(taskID)

	if err != nil {
		msg := fmt.Sprintf("[manager-api] error parsing UUID %s: %v", taskID, err)

		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	_, exists := a.Manager.TaskDb[taskUUID]

	if !exists {
		msg := fmt.Sprintf("[manager-api] no task with ID %v found", taskUUID)

		log.Println(msg)
		http.Error(w, msg, http.StatusNotFound)

		return
	}

	taskToStop := *a.Manager.TaskDb[taskUUID]
	taskToStop.State = task.Completed

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now(),
		Task:      taskToStop,
	}

	a.Manager.AddTask(te)

	log.Printf("[manager-api] added task event %v to stop task %v", te.ID, taskToStop.ID)

	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("stopping task"))
}
