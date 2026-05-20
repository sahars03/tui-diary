package main

// packages for formatting (e.g. printing) and interacting with the operating system
import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	fmt.Println(" ###### TUI DIARY ###### ")
	fmt.Println("Welcome!")
	printHelp()

	// channel for receiving operating system signals, with a buffer size of one item
	sigChan := make(chan os.Signal, 1)
	// connects channel to OS signals
	// when either signal occurs (interrupt and sigterm), it is sent to the channel sigChan
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	scanner := bufio.NewScanner(os.Stdin)
	inputChan := make(chan string)

	go func() {
		for scanner.Scan() {
			inputChan <- strings.TrimSpace(scanner.Text())
		}
	}()

	fmt.Print("> ")
	for {
		select {
		case <-sigChan:
			fmt.Println("\nBye!")
			return
		case input := <-inputChan:
			handleCommand(input, inputChan)
			fmt.Print("> ")
		}
	}
}

func printHelp() {
	fmt.Println("########################################################")
	fmt.Println("# Press Ctrl+C at any time to quit.")
	fmt.Println("# Commands:")
	fmt.Println("#  [n]ew - write a new entry")
	fmt.Println("#")
	fmt.Println("# Enter [h]elp to make these instructions reappear!    #")
	fmt.Println("########################################################")
}

func handleCommand(input string, inputChan chan string) {
	switch strings.ToLower(input) {
	case "h", "help":
		printHelp()
	case "n", "new":
		writeNewEntry(inputChan)
	default:
		fmt.Printf("Unknown command: %q. Try: [n]ew\n", input)
	}
}

func writeNewEntry(inputChan chan string) {
	fmt.Println("--- NEW ENTRY ---")
	fmt.Println("Write your entry (press Enter twice when done):")

	var lines []string

	for input := range inputChan {
		if input == "" && len(lines) > 0 {
			break
		}
		lines = append(lines, input)
	}

	if len(lines) == 0 {
		fmt.Println("(empty entry discarded)")
		return
	}

	entry := strings.Join(lines, "\n")
	fmt.Printf("Entry saved (%d lines)!\n", len(lines))
	_ = entry
}