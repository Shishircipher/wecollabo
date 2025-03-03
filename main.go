package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

// Global logger (for demonstration, replace with structured logger like Zap or Logrus)
var logger = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)

// Panic recovery middleware
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				var buf bytes.Buffer
				pprof.Lookup("goroutine").WriteTo(&buf, 2) // Capture stack trace
				logger.Printf("PANIC RECOVERED: %v\nSTACK TRACE:\n%s", err, buf.String())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Simulated database cleanup
func cleanup() {
	logger.Println("Cleaning up resources (DB, caches, etc.)...")
	time.Sleep(2 * time.Second) // Simulate cleanup time
	logger.Println("Cleanup complete. Server shutting down.")
}

func main() {
	// HTTP server setup
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

	// Simulating an endpoint that causes a panic
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Something went wrong!") // Simulated panic
	})

	// Wrap middleware to recover from panics
	handler := recoveryMiddleware(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// Channel to listen for OS signals (Ctrl+C, kill command)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		logger.Println("Server started on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown handling
	<-stop // Wait for termination signal
	logger.Println("Shutdown signal received. Cleaning up...")

	// Set timeout for cleanup operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Defer panic recovery & logging before shutdown
	defer func() {
		if x := recover(); x != nil {
			var buf bytes.Buffer
			pprof.Lookup("goroutine").WriteTo(&buf, 2)
			logger.Printf("PANIC DURING SHUTDOWN: %v\nSTACK TRACE:\n%s", x, buf.String())
		}
	}()

	// Perform cleanup (close DB, stop workers, etc.)
	cleanup()

	// Gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server shutdown failed: %v", err)
	}
	logger.Println("Server exited gracefully.")
}

