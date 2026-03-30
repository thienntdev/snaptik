package logger

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
)

// Init configures the global log package from charmbracelet/log
func Init(isProd bool) {
	// Configure the default logger
	log.SetOutput(os.Stderr)
	log.SetReportCaller(true) // Highlights the calling file and line number
	log.SetTimeFormat(time.RFC3339)

	if isProd {
		log.SetLevel(log.InfoLevel)
		// For production, maybe we want JSON instead of text, but text is fine if requested for readability
		log.SetFormatter(log.TextFormatter) 
	} else {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(log.TextFormatter)
	}

	// Make the standard Go logger use our new logger
	log.SetDefault(log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Level:           log.DebugLevel,
	}))
}
