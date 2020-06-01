package goredis

import (
	"testing"
	"time"
)

const testPrefix = "goredis"

func TestService_Get(t *testing.T) {
	p := CreatePool(testHost, testPort, testDb, testPassword)
	defer p.Close()
	s := NewService(p, testPrefix)
	key := "username"
	value := "hello"
	if err := s.Set(key, value, 1*time.Minute); err != nil {
		t.FailNow()
	}
	var v string
	if err := s.Get(key, &v); err != nil {
		t.FailNow()
	}
	t.Log("v", v)
	if v != value {
		t.FailNow()
	}
	if err := s.Delete(key); err != nil {
		t.FailNow()
	}
}
