package senddat

import (
	"log/slog"
	"os"
)

func init() {
	// set logging level to debugging if required.
	if os.Getenv("DEBUG") == "1" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	// set the wait multiplier to 0 to avoid delaying tests.
	gWaitMultiplier = 0

	// prevent the test strings output to the terminal during tests.
	null, err := os.Open(os.DevNull)
	if err != nil {
		slog.Error("UNABLE TO OPEN NULL DEVICE")
	} else {
		SenddatOutput = null
	}

}
