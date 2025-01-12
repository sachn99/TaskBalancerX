package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// Task represents a task with a file and metadata
type Task struct {
	ID   string `json:"id"`
	File string `json:"file"`
}

// TaskQueue is a channel that holds tasks
var TaskQueue = make(chan Task, 100)

// WorkerURL is the URL of the Rust worker nodes
const WorkerURL = "http://localhost:8081/process"

func main() {
	router := mux.NewRouter()

	// Define the endpoint to accept tasks
	router.HandleFunc("/tasks", CreateTaskHandler).Methods("POST")

	go StartWorker()

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// CreateTaskHandler handles incoming tasks
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task Task

	// Read the multipart form data
	err := r.ParseMultipartForm(10 << 20) // limit to 10 MB files
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get file from the form data
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save the file temporarily
	tempFile, err := ioutil.TempFile(os.TempDir(), "upload-*.jpg")
	if err != nil {
		http.Error(w, "Unable to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	// Write the file to the temporary location
	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	// Create a new task
	task = Task{
		ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		File: tempFile.Name(),
	}

	// Send the task to the queue
	TaskQueue <- task

	// Respond to the client
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(task)
}

// StartWorker starts a worker that processes tasks
func StartWorker() {
	for {
		select {
		case task := <-TaskQueue:
			ProcessTask(task)
		}
	}
}

// ProcessTask sends the task to the Rust worker node
func ProcessTask(task Task) {
	// Prepare the payload as JSON
	payload, err := json.Marshal(task)
	if err != nil {
		log.Printf("Error marshaling task: %v", err)
		return
	}

	resp, err := http.Post(WorkerURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error sending task to worker: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Worker failed to process task: %s", task.ID)
	}
	log.Printf("Task %s processed successfully", task.ID)
}
