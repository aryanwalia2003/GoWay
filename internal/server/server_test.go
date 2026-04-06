package server_test

import (
	"net/http"
	"syscall"
	"testing"
	"time"

	"awb-gen/internal/server"
)

// TestGracefulShutdown_ActiveJobCompletes ensures an active request finishes before the server shuts down.
func TestGracefulShutdown_ActiveJobCompletes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Simulated active job
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("done"))
	})

	srv := server.NewServer("127.0.0.1:18482", mux)

	go func() {
		_ = srv.Start()
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	errCh := make(chan error, 1)
	go func() {
		// Make a request that takes 200ms
		resp, err := http.Get("http://" + srv.Addr() + "/slow")
		if err != nil {
			errCh <- err
			return
		}
		resp.Body.Close()
		errCh <- nil
	}()

	// Wait a bit, then send SIGTERM while the request is still active
	time.Sleep(50 * time.Millisecond)

	shutdownStartTime := time.Now()
	srv.Signal(syscall.SIGTERM) // Trigger graceful shutdown

	// Wait for Start() to return
	srv.Wait()
	shutdownDuration := time.Since(shutdownStartTime)

	if err := <-errCh; err != nil {
		t.Fatalf("Expected active request to complete, got error: %v", err)
	}

	if shutdownDuration < 150*time.Millisecond {
		t.Fatalf("Server shut down too quickly, didn't wait for active job. Duration: %v", shutdownDuration)
	}
}

// TestGracefulShutdown_TimeoutBreached ensures server force-exits if 15s deadline passes.
func TestGracefulShutdown_TimeoutBreached(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/infinite", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Second) // Block beyond 15s deadline
	})

	// Inject a 1-second deadline instead of 15 for faster testing
	srv := server.NewServerWithTimeout("127.0.0.1:18483", mux, 1*time.Second)

	go func() {
		_ = srv.Start()
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	go func() {
		// This will block
		_, _ = http.Get("http://" + srv.Addr() + "/infinite")
	}()

	time.Sleep(50 * time.Millisecond)

	shutdownStartTime := time.Now()
	srv.Signal(syscall.SIGTERM)
	srv.Wait()
	shutdownDuration := time.Since(shutdownStartTime)

	if shutdownDuration < 1*time.Second || shutdownDuration > 2*time.Second {
		t.Fatalf("Server should force shutdown around the 1s deadline. Actual duration: %v", shutdownDuration)
	}
}
