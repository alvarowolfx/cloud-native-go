package telemetry

import (
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
)

func InitLogger() {
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "json" {
		log.SetHandler(json.New(os.Stderr))
	}
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		log.SetLevel(log.MustParseLevel(logLevel))
	}
}
