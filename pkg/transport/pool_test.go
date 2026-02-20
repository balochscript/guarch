package transport

import (
	"testing"
)

func TestPoolCreate(t *testing.T) {
	pool := NewPool("127.0.0.1:9999", 5)

	if pool.Size() != 0 {
		t.Errorf("initial size = %d want 0", pool.Size())
	}

	pool.Close()

	t.Log("OK: pool created and closed")
}

func TestPoolSize(t *testing.T) {
	pool := NewPool("127.0.0.1:9999", 3)

	if pool.maxSize != 3 {
		t.Errorf("maxSize = %d want 3", pool.maxSize)
	}
	if pool.maxRetry != 3 {
		t.Errorf("maxRetry = %d want 3", pool.maxRetry)
	}

	pool.Close()

	t.Log("OK: pool config correct")
}
