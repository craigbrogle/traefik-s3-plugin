package local

import (
	"fmt"
	"net/http"
	"os"

	"github.com/craigbrogle/traefik-s3-plugin/log"
)

type Local struct {
	directory string
}

func New(directory string) *Local {
	return &Local{
		directory: directory,
	}
}

func (local *Local) Get(name string, _ http.ResponseWriter) ([]byte, error) {
	filePath := fmt.Sprintf("%s/%s", local.directory, name)
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	log.Debug(fmt.Sprintf("%q read", filePath))
	return content, nil
}
