package main

import (
	"os"
	"os/signal"
)

// Infinite intercepts all signals but does nothing with them.
// To kill the process we must send a SIGKILL signal, that can not be handled.
func main() {
	sigs := make(chan os.Signal)

	signal.Notify(sigs)

	done := make(chan struct{})

	<-done
}
