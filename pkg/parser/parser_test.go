package parser

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLoadMessagesResilience(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "config"), 0o755); err != nil {
		t.Fatalf("create config directory: %v", err)
	}
	dummyContent := `- name: test
  text: "Hello World"
`
	if err := os.WriteFile(filepath.Join(tempDir, "config", "messages.yml"), []byte(dummyContent), 0o644); err != nil {
		t.Fatalf("write messages fixture: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	previousMessages := messagesMap
	messagesMap = make(map[string]Message)
	loadOnce = sync.Once{}
	t.Cleanup(func() {
		_ = os.Chdir(workingDir)
		messagesMap = previousMessages
		loadOnce = sync.Once{}
	})

	text, _ := GetMessageTelego("test", nil)
	if text != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", text)
	}
}
