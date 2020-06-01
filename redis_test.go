package goredis

import "testing"

const (
	testHost     = "127.0.0.1"
	testPort     = 6379
	testPassword = ""
	testDb       = 1
)

func TestCreatePool(t *testing.T) {
	p := CreatePool(testHost, testPort, testDb, testPassword)
	if _, err := p.Get().Do("PING"); err != nil {
		t.FailNow()
	}
	p.Close()
}
