// Command testservice is a minimal HTTP server used as a test fixture
// for the service lifecycle tests in the testastic package.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var buildStamp = "default-build-stamp"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Fprintln(os.Stderr, "PORT environment variable is required")
		os.Exit(1)
	}

	if os.Getenv("EXIT_EARLY") == "true" {
		fmt.Fprintln(os.Stderr, "exiting early as requested")
		os.Exit(1)
	}

	if d := os.Getenv("SLOW_START"); d != "" {
		dur, err := time.ParseDuration(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid SLOW_START duration: %v\n", err)
			os.Exit(1)
		}

		time.Sleep(dur)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /env", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, os.Getenv("TESTASTIC_TEST_VALUE"))
	})
	mux.HandleFunc("GET /args", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(os.Args[1:])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("GET /build-info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, buildStamp)
	})
	mux.HandleFunc("GET /cwd", func(w http.ResponseWriter, _ *http.Request) {
		wd, err := os.Getwd()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, wd)
	})
	mux.HandleFunc("GET /data", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := map[string]string{
			"message": "hello",
			"version": "1.0.0",
		}

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	server := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "listen error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
	}
}
