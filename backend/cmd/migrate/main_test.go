package main

import "testing"

func TestRedactConnectionErrorRemovesCredentials(t *testing.T) {
	message := redactConnectionError("failed to connect: postgresql://postgres.project:super-secret@pooler.supabase.com:5432/postgres")
	if message == "" || contains(message, "super-secret") {
		t.Fatalf("connection error leaked credentials: %q", message)
	}
}

func contains(value, fragment string) bool {
	return len(fragment) == 0 || (len(value) >= len(fragment) && find(value, fragment))
}

func find(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
