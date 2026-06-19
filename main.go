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
			fmt.Print("\nBye!")
			return
		case input := <-inputChan:
			keepGoing := handleCommand(input, inputChan, conn)
			if !keepGoing {
				fmt.Print("Bye!")
				return
			}
			fmt.Print("> ")
		}
	}
}

func printHelp() {
	fmt.Println("########################################################")
	fmt.Println("# Commands:")
	fmt.Println("#  [n]ew - write a new entry")
	fmt.Println("#  [a]ll - list all of the entries you have made")
	fmt.Println("#  [v]iew - take a look at a specific entry")
	fmt.Println("#  [d]elete - delete a specific entry")	
	fmt.Println("#  [c]lear - delete all of your entries")	
	fmt.Println("#  [q]uit - leave the application")	
	fmt.Println("#")
	fmt.Println("# Enter [h]elp to make these instructions reappear!    #")
	fmt.Println("########################################################")
}

func handleCommand(input string, inputChan chan string, conn *pgx.Conn) bool {
	switch strings.ToLower(input) {
	case "h", "help":
		printHelp()
	case "n", "new":
		writeNewEntry(inputChan, conn)
	case "a", "all":
		listEntries(conn)
	case "v", "view":
		displayEntry(inputChan, conn)
	case "d", "delete":
		deleteEntry(inputChan, conn)
	case "c", "clear":
		deleteAllEntries(inputChan, conn)
	case "q", "quit":
		return false
	default:
		fmt.Printf("Unknown command: %q. Try something else.\n", input)
	}
	return true
}

// TODO: implement cancellation so that when a user does not want to save their entry they cancel writing it
func writeNewEntry(inputChan chan string, conn *pgx.Conn) {
	fmt.Println("--- NEW ENTRY ---")
	fmt.Println("# press Enter twice when done")
	fmt.Println("# type :cancel and press Enter to discard the entry")
	fmt.Println("Write your entry below:")

	var lines []string
	for input := range inputChan {
		if input == ":cancel" {
			fmt.Println("Entry cancelled.")
			return
		}
		if input == "" && len(lines) > 0 {
			break
		}
		lines = append(lines, input)
	}

	contents := strings.TrimSpace(strings.Join(lines, "\n"))
	if contents == "" {
		fmt.Println("(empty entry discarded)")
		return
	}

	if err := saveEntry(conn, contents); err != nil {
		fmt.Println("Error saving entry:", err)
		return
	}
}

// TODO: message where if there are no existing entries the user is told that before they start inputting IDs
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

func deleteEntry(inputChan chan string, conn *pgx.Conn) {
	fmt.Println("Enter the ID of the entry you want to delete:")

	input := <-inputChan

	id, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Invalid ID — please enter a number. Type [d]elete to try again.")
		return
	}

	entry, err := loadOneEntry(conn, id)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("This is the entry you wish to delete:")
	fmt.Printf("--- Entry #%d | %s ---\n", entry.ID, entry.Date.Format("2 Jan 2006 15:04"))
	fmt.Println(entry.Contents)
	fmt.Println("---")	

	fmt.Println("Are you sure you want to delete this entry? Type [y]es to delete and anything else to cancel")

	input = <-inputChan
	
	if strings.ToLower(input) == "y" || strings.ToLower(input) == "yes" {
		if err := deleteOneEntry(conn, id); err != nil {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Println("Entry deletion cancelled.")
	}
}

func deleteAllEntries(inputChan chan string, conn *pgx.Conn) {

	fmt.Println("Are you sure you want to delete all entries? This cannot be undone.\nType [y]es to delete and anything else to cancel")

	input := <-inputChan
	
	if strings.ToLower(input) == "y" || strings.ToLower(input) == "yes" {
		if err := clearEntries(conn); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("All entries have been deleted.")
	} else {
		fmt.Println("Deletion of entries cancelled.")
	}
}