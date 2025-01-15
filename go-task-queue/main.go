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
	"context"
	"sync"
    "syscall"
    "os/signal"

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


var TaskStatus = sync.Map{}

func main() {
	router := mux.NewRouter()

	// Define the endpoint to accept tasks
	router.HandleFunc("/tasks", CreateTaskHandler).Methods("POST")

    router.HandleFunc("/status", TaskStatusHandler).Methods("GET")
	router.HandleFunc("/health", HealthCheckHandler).Methods("GET")

	//go StartWorker()

	// Context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    var wg sync.WaitGroup

    wg.Add(1)
    go func() {
        defer wg.Done()
    	StartWorker(ctx)
    }()


	server := &http.Server{Addr: ":8080", Handler: router}

	// Handle OS signals for graceful shutdown
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		log.Println("Shutting down server...")
		cancel()          // Cancel worker context
		server.Close()    // Stop the HTTP server
	}()

	log.Println("Starting server on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
    		log.Fatalf("Server error: %v", err)
    }

    wg.Wait() // Wait for all goroutines to finish
    log.Println("Server stopped gracefully.")

	//log.Fatal(http.ListenAndServe(":8080", router))
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

	TaskStatus.Store(task.ID, "queued")

	// Respond to the client
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(task)
}

func TaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if status, ok := TaskStatus.Load(taskID); ok {
		json.NewEncoder(w).Encode(map[string]string{"status": status.(string)})
	} else {
		http.Error(w, "Task not found", http.StatusNotFound)
	}
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// StartWorker starts a worker that processes tasks
func StartWorker() {
	for {
		select {
		case task := <-TaskQueue:
			TaskStatus.Store(task.ID, "processing")
			ProcessTask(task)
			TaskStatus.Store(task.ID, "completed")
		case <-ctx.Done():
			log.Println("Worker stopped")
			return
		}
	}
}

// ProcessTask sends the task to the Rust worker node
func ProcessTask(task Task) {
    for retries := 0; retries < 3; retries++ {
    	// Prepare the payload as JSON

        payload, err := json.Marshal(task)
        if err != nil {
            log.Printf("Error marshaling task: %v", err)
            return
        }

        resp, err := http.Post(WorkerURL, "application/json", bytes.NewBuffer(payload))
        if err != nil {
            log.Printf("Error sending task to worker: %v", err)
            continue // Retry logic
        }
        defer resp.Body.Close() // Ensures the response body is always closed

        if resp.StatusCode == http.StatusOK {
            log.Printf("Task %s processed successfully", task.ID)
            return
        }

        log.Printf("Retrying task %s (%d/3)", task.ID, retries+1)
        time.Sleep(2 * time.Second)
    }

    log.Printf("Task %s failed after retries", task.ID)
}

