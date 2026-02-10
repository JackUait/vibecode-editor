package util_test

import (
	"testing"

	"github.com/jackuait/ghost-tab/internal/util"
)

func TestOutputJSON(t *testing.T) {
	data := map[string]interface{}{
		"name":     "test-project",
		"path":     "/home/user/test",
		"selected": true,
	}

	output, err := util.OutputJSON(data)
	if err != nil {
		t.Fatalf("OutputJSON failed: %v", err)
	}

	expected := `{"name":"test-project","path":"/home/user/test","selected":true}`
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestOutputJSONError(t *testing.T) {
	data := map[string]interface{}{
		"invalid": make(chan int), // channels can't be marshaled
	}

	_, err := util.OutputJSON(data)
	if err == nil {
		t.Error("Expected error for invalid data, got nil")
	}
}
