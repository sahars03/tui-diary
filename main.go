package main

// packages for formatting (e.g. printing) and interacting with the operating system
import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("This is the start of my TUI app...")
	fmt.Println("Ctrl+C to quit.")

	// channel for receiving operating system signals, with a buffer size of one item
	sigChan := make(chan os.Signal, 1)
	// connects channel to OS signals
	// when either signal occurs (interrupt and sigterm), it is sent to the channel sigChan
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// program execution is paused until sigChan receives a signal
	<-sigChan

	fmt.Println("\nBye!")
}
