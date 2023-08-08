package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/dkartachov/cube/task"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	var te task.TaskEvent

	err := d.Decode(&te)

	if err != nil {
		msg := fmt.Sprintf("[worker-api: %s] error unmarshalling body: %v", a.Worker.Name, err)
		log.Print(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	a.Worker.AddTask(te.Task)

	log.Printf("[worker-api: %s] added task %v", a.Worker.Name, te.Task.ID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	if taskID == "" {
		msg := fmt.Sprintf("[worker-api: %s] missing taskID in request", a.Worker.Name)

		log.Print(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	taskUUID, err := uuid.Parse(taskID)

	if err != nil {
		msg := fmt.Sprintf("[worker-api: %s] error parsing UUID %s: %v", a.Worker.Name, taskID, err)

		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	_, exists := a.Worker.Db[taskUUID]

	if !exists {
		msg := fmt.Sprintf("[worker-api: %s] no task with ID %v found", a.Worker.Name, taskUUID)

		log.Println(msg)
		http.Error(w, msg, http.StatusNotFound)

		return
	}

	taskToStop := a.Worker.Db[taskUUID]
	taskToStopCopy := *taskToStop
	taskToStopCopy.State = task.Completed

	a.Worker.AddTask(taskToStopCopy)

	log.Printf("[worker-api: %s] added task %v to stop container %v", a.Worker.Name, taskToStopCopy.ID, taskToStopCopy.ContainerId)

	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.Stats)
}
