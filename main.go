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
	"strconv"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {

    if err := godotenv.Load(".env.local"); err != nil {
        fmt.Println("Warning: no .env file found")
    }

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
	// TODO: change ctrl-c to [q]uit
	fmt.Println("########################################################")
	fmt.Println("# Enter [q]uit at any time to quit.")
	fmt.Println("# Commands:")
	fmt.Println("#  [n]ew - write a new entry")
	fmt.Println("#  [a]ll - list all of the entries you have made")
	fmt.Println("#  [v]iew - take a look at a specific entry")
	fmt.Println("#  [c]lear - delete all of your entries")	
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
	case "v", "view":
		displayEntry(inputChan, conn)
	default:
		fmt.Printf("Unknown command: %q. Try something else.\n", input)
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
}

func displayEntry(inputChan chan string, conn *pgx.Conn) {
	fmt.Println("Enter the ID of the entry you want to read:")

	input := <-inputChan

	id, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Invalid ID — please enter a number. Type [v]iew to try again.")
		return
	}

	entry, err := loadOneEntry(conn, id)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("--- Entry #%d | %s ---\n", entry.ID, entry.Date.Format("2 Jan 2006 15:04"))
	fmt.Println(entry.Contents)
	fmt.Println("---")
}