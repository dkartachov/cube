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
		msg := fmt.Sprintf("Error unmarshalling body: %v", err)
		log.Print(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	a.Worker.AddTask(te.Task)

	log.Printf("Added task %v", te.Task.ID)

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
		msg := "missing taskID in request"

		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	taskUUID, err := uuid.Parse(taskID)

	if err != nil {
		msg := fmt.Sprintf("error parsing UUID %s: %v", taskID, err)

		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	_, exists := a.Worker.Db[taskUUID]

	if !exists {
		msg := fmt.Sprintf("No task with ID %v found", taskUUID)

		log.Println(msg)
		http.Error(w, msg, http.StatusNotFound)

		return
	}

	taskToStop := a.Worker.Db[taskUUID]
	taskToStopCopy := *taskToStop
	taskToStopCopy.State = task.Completed

	a.Worker.AddTask(taskToStopCopy)

	log.Printf("Added task %v to stop container %v", taskToStopCopy.ID, taskToStopCopy.ContainerId)

	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.Stats)
}
