package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"awb-gen/internal/middleware"
)

func TestConcurrency_RequestsQueueNotRejected(t *testing.T) {
	limit := 2
	mw := middleware.NewConcurrency(limit)
	var active int
	var maxActive int
	var mu sync.Mutex

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond) // Simulate work

		mu.Lock()
		active--
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))

	reqCount := 5
	var wg sync.WaitGroup
	wg.Add(reqCount)

	for i := 0; i < reqCount; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code == http.StatusTooManyRequests {
				t.Errorf("expected request to queue, got 429 Too Many Requests")
			}
			if rr.Code != http.StatusOK {
				t.Errorf("expected 200 OK, got %d", rr.Code)
			}
		}()
	}

	wg.Wait()

	if maxActive > limit {
		t.Errorf("max active requests %d exceeded limit %d", maxActive, limit)
	}
}

func TestConcurrency_SlotReleasedAfterCompletion(t *testing.T) {
	mw := middleware.NewConcurrency(1)
	var active int32

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&active, 1)
		defer atomic.AddInt32(&active, -1)
		w.WriteHeader(http.StatusOK)
	}))

	// Sequential calls should both succeed and not block
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr1.Code != http.StatusOK || rr2.Code != http.StatusOK {
		t.Errorf("expected 200 OK for both, got %d and %d", rr1.Code, rr2.Code)
	}
}

func TestConcurrency_SlotReleasedOnError(t *testing.T) {
	mw := middleware.NewConcurrency(1)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Fail") == "true" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("Fail", "true")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr1.Code)
	}
	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr2.Code)
	}
}

func TestConcurrency_QueuedJobCompletesOnShutdown(t *testing.T) {
	mw := middleware.NewConcurrency(1)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done(): // If the context was prematurely cancelled, this triggers
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	var wg sync.WaitGroup
	wg.Add(2)

	serverExitCtx, cancelServer := context.WithCancel(context.Background())

	// Start a job
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("job 1 did not complete successfully")
		}
	}()

	// Start a queued job
	time.Sleep(10 * time.Millisecond) // Ensures job 1 occupies the slot
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(serverExitCtx) // Shutdown signal propagates to client
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}()

	time.Sleep(20 * time.Millisecond)
	cancelServer() // Simulate shutdown signal

	// We wait to ensure it all unblocks
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("queued job deadlocked during shutdown")
	}
}
