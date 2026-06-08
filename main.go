package main

// packages for formatting (e.g. printing) and interacting with the operating system
import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"context"
	"github.com/jackc/pgx/v5"
)

func main() {
	fmt.Println(" ###### TUI DIARY ###### ")
	fmt.Println("Welcome!")
	printHelp()

	conn, err := connect()
	if err != nil {
		fmt.Println("Error connecting to database:", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	if err := setupDB(conn); err != nil {
		fmt.Println("Error setting up database:", err)
		os.Exit(1)
	}

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
			handleCommand(input, inputChan, conn)
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

func handleCommand(input string, inputChan chan string, conn *pgx.Conn) {
	switch strings.ToLower(input) {
	case "h", "help":
		printHelp()
	case "n", "new":
		writeNewEntry(inputChan, conn)
	case "a", "all":
		listEntries(conn)
	default:
		fmt.Printf("Unknown command: %q. Try something else.", input)
	}
}

func writeNewEntry(inputChan chan string, conn *pgx.Conn) {
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

	if err := saveEntry(conn, strings.Join(lines, "\n")); err != nil {
		fmt.Println("Error saving entry:", err)
		return
	}
	fmt.Println("Entry saved!")
}