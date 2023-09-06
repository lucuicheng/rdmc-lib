package main

import (
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	"testing"
)

func TestSqliteRequest3(t *testing.T) {
	tests := []struct {
		key    string
		source interface{}
		want   bool
	}{
		{"normal", 10000000.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			//manager.DatabaseRequest("localhost")
		})
	}
}
