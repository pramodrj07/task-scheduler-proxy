package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Configuration constants
const (
	maxWorkers   = 5               // Maximum number of concurrent workers
	queueSize    = maxWorkers      // Size of the worker queue (matches worker count)
	requestDelay = 5 * time.Second // Artificial delay to simulate rate limiting
)

// Job represents a unit of work to be processed by a worker
type Job struct {
	ID       string      // Unique identifier for the job
	Response chan string // Channel to send the result back to the requester
}

// Worker represents a worker that can process jobs
type Worker struct {
	ID       int      // Unique identifier for the worker
	JobQueue chan Job // Channel to receive jobs for processing
}

// Start initializes the worker and begins listening for jobs
// It registers itself as available and processes jobs until context is cancelled
func (w *Worker) Start(ctx context.Context, wg *sync.WaitGroup, available chan *Worker) {
	defer wg.Done() // Signal completion when worker shuts down

	for {
		// Check if shutdown signal received
		select {
		case <-ctx.Done():
			log.Printf("Worker %d received shut down signal", w.ID)
			return
		default:
			// Register this worker as available for new jobs
			available <- w
		}

		// Wait for a job or shutdown signal
		select {
		case <-ctx.Done():
			log.Printf("Worker %d received shut down signal", w.ID)
			return
		default:
			// Block until a job is received
			job := <-w.JobQueue
			log.Printf("Worker %d processing job %s", w.ID, job.ID)

			// Simulate processing time (rate limiting)
			time.Sleep(requestDelay)

			// Send the processing result back to the requester
			job.Response <- fmt.Sprintf("processed by go-routine-%d", w.ID)
		}
	}
}

func main() {
	var wg sync.WaitGroup

	// Channel to track available workers (acts as a semaphore)
	availableWorkers := make(chan *Worker, maxWorkers)

	// Context for graceful shutdown coordination
	ctx, cancel := context.WithCancel(context.Background())
	server := &http.Server{Addr: ":8080"}

	// Create and start worker pool
	for i := 1; i <= maxWorkers; i++ {
		worker := Worker{
			ID:       i,
			JobQueue: make(chan Job), // Each worker has its own job queue
		}
		wg.Add(1) // Add to wait group for graceful shutdown

		// Start worker in separate goroutine
		go worker.Start(ctx, &wg, availableWorkers)
	}

	// HTTP endpoint to process requests
	http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		// Try to get an available worker (non-blocking)
		select {
		case worker := <-availableWorkers:
			// Create a new job with unique ID and response channel
			job := Job{
				ID:       fmt.Sprintf("req-%d", time.Now().UnixNano()),
				Response: make(chan string),
			}

			// Send job to the worker (non-blocking to avoid deadlock)
			go func() {
				worker.JobQueue <- job
			}()

			// Wait for the worker to complete the job and send response
			result := <-job.Response
			w.Write([]byte(result))
		default:
			// No workers available - return error
			http.Error(w, "No workers available", http.StatusServiceUnavailable)
		}
	})

	// HTTP endpoint to gracefully shutdown the server
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Shut down requested via /shutdown API")
		w.Write([]byte("Shutting Down"))

		// Perform shutdown in separate goroutine to avoid blocking the response
		go func() {
			cancel() // Cancel context to signal all workers to stop

			// Gracefully shutdown the HTTP server
			if err := server.Shutdown(ctx); err != nil {
				log.Printf("Unable to shut down the server: %v", err)
			}
		}()
	})

	// Start the HTTP server
	log.Println("Server starting on :8080")
	log.Fatal(server.ListenAndServe())

	// Wait for all workers to complete (this line is unreachable due to log.Fatal above)
	wg.Wait()
}
