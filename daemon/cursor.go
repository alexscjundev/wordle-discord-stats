package daemon

import (
	"os"
	"strings"
)

type MessageCursor interface {
	Get() (string, error)
	Set(id string) error
}

type FileCursor struct {
	path string
}

func NewFileCursor(path string) *FileCursor {
	return &FileCursor{path: path}
}

func (f *FileCursor) Get() (string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (f *FileCursor) Set(id string) error {
	return os.WriteFile(f.path, []byte(id), 0644)
}
