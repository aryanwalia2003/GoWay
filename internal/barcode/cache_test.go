package barcode

import (
	"bytes"
	"sync"
	"testing"
)

func TestCache_GetSet(t *testing.T) {
	c := NewCache()

	val1 := "AWB123"
	data1 := []byte("fake-png-data")

	// Test Miss
	if _, ok := c.Get(val1); ok {
		t.Error("Expected Get to return false for missing key")
	}

	// Test Set
	c.Set(val1, data1)

	// Test Hit
	if cached, ok := c.Get(val1); !ok || !bytes.Equal(cached, data1) {
		t.Error("Expected Get to return the cached data successfully")
	}

	// Test concurrency
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get(val1)
			c.Set("AWB-concurrent", []byte("data"))
		}()
	}
	wg.Wait()
}
