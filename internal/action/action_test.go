package action

import "testing"

func TestNew(t *testing.T) {
	code := `package main
	import "fmt"

	func main() {
		fmt.Println("hello")
	}`

	action := New(code)

	if action == nil {
		t.Fatal("expected action, got nil")
	}

	if action.Code != code {
		t.Fatalf("expected %q, got %q", code, action.Code)
	}
}
