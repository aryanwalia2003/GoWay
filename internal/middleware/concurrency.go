package middleware

import (
	"net/http"
)

// Concurrency middleware limits active generation tasks.
type Concurrency struct {
	sem chan struct{} ///this is a semaphore  channel
}

// NewConcurrency initializes a new concurrency middleware with a specific max limit.
func NewConcurrency(max int) *Concurrency {
	return &Concurrency{
		sem: make(chan struct{}, max), //it is a buffered channel where buff limit is max
	}
}

// Wrap applies the concurrency limit to the given handler.
func (c *Concurrency) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case c.sem <- struct{}{}: //if the channel is not full then it will send the value and proceed
			defer func() { <-c.sem }() //after the function completes it will remove the value from the channel
		case <-r.Context().Done(): //if the channel is full then it will wait for the value to be removed from the channel
			w.WriteHeader(http.StatusInternalServerError) // Client disconnected / shutdown
			return
		}
		next.ServeHTTP(w, r)
	})
}
