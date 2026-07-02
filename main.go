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
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"

)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#e49b5b")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#1a8726")).
			Padding(1, 2)

	idStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	dateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	commandStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#d8db7f"))

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f9c1c1")).
		Bold(true)

	entryStyle = lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("212")).
		PaddingLeft(2)

	entryTextStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#c679e4"))

	cancelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fa3f3f"))

	warningStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#fe9696"))
)

func main() {

    if err := godotenv.Load(".env.local"); err != nil {
        fmt.Println("Warning: no .env file found")
    }

	figure.NewFigure("TUI DIARY", "", true).Print()
	fmt.Println()
	fmt.Println(titleStyle.Render("                          Welcome!"))
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
		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading input:", err)
		}
		close(inputChan)
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
	help := lipgloss.NewStyle().Bold(true).Render("Commands:") + "\n" +
		"  " + commandStyle.Render("[n]ew") + "    write a new entry\n" +
		"  " + commandStyle.Render("[a]ll") + "    list all of the entries you have made\n" +
		"  " + commandStyle.Render("[v]iew") + "   take a look at a specific entry\n" +
		"  " + commandStyle.Render("[d]elete") + " delete a specific entry\n" +
		"  " + commandStyle.Render("[c]lear") + "  delete all of your entries\n" +
		"  " + commandStyle.Render("[q]uit") + "   leave the application\n\n" +
		"Enter " + commandStyle.Render("[h]elp") + " to make these instructions reappear!"

	fmt.Println(boxStyle.Render(help))
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

func writeNewEntry(inputChan chan string, conn *pgx.Conn) {

	one := entryTextStyle.Render("NEW ENTRY")
	two := "   press " + commandStyle.Render("Enter ") + lipgloss.NewStyle().Underline(true).Render("twice") + " when done"
	three := "   type " + commandStyle.Render(":cancel") + " and press " + commandStyle.Render("Enter") + " to discard the entry"
	four := entryTextStyle.Render("Write your entry below:")

	content := one + "\n" + two + "\n" + three + "\n" + four

	fmt.Println(entryStyle.Render(content))

	var lines []string
	for input := range inputChan {
		if input == ":cancel" {
			fmt.Println(cancelStyle.Render("Entry cancelled."))
			return
		}
		if input == "" && len(lines) > 0 {
			break
		}
		lines = append(lines, input)
	}

	contents := strings.TrimSpace(strings.Join(lines, "\n"))

	if contents == "" {
		fmt.Println(cancelStyle.Render("Empty entry discarded."))
		return
	}

	if err := saveEntry(conn, contents); err != nil {
		fmt.Println("Error saving entry:", err)
		return
	}
}

func displayEntry(inputChan chan string, conn *pgx.Conn) {
	fmt.Print(promptStyle.Render("Enter the ") + idStyle.Render("ID ") + promptStyle.Render("of the entry you want to read: "))

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

	header := idStyle.Render(fmt.Sprintf("#%d", entry.ID)) + "  " +
		dateStyle.Render(entry.Date.Format("2 Jan 2006 15:04"))

	content := header + "\n\n" + entry.Contents

	fmt.Println(entryStyle.Render(content))
}

func deleteEntry(inputChan chan string, conn *pgx.Conn) {
	fmt.Print(promptStyle.Render("Enter the ") + idStyle.Render("ID ") + promptStyle.Render("of the entry you want to delete: "))

	input := <-inputChan

	id, err := strconv.Atoi(input)
	if err != nil {
			fmt.Println(cancelStyle.Render("Invalid ID — please enter a number. Type [d]elete to try again."))
		return
	}

	entry, err := loadOneEntry(conn, id)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(promptStyle.UnsetBold().Foreground(lipgloss.Color("#fe9696")).Render("This is the entry you wish to delete:\n"))
	// fmt.Printf("--- Entry #%d | %s ---\n", entry.ID, entry.Date.Format("2 Jan 2006 15:04"))
	// fmt.Println(entry.Contents)
	// fmt.Println("---")	

	header := idStyle.Render(fmt.Sprintf("#%d", entry.ID)) + "  " +
		dateStyle.Render(entry.Date.Format("2 Jan 2006 15:04"))

	content := header + "\n\n" + entry.Contents

	fmt.Println(entryStyle.Render(content))

	fmt.Print(warningStyle.Render("\nAre you sure you want to delete this entry? Type ") + commandStyle.Italic(true).Render("[y]es") + warningStyle.Render(" to delete it and anything else to cancel: "))

	input = <-inputChan
	
	if strings.ToLower(input) == "y" || strings.ToLower(input) == "yes" {
		if err := deleteOneEntry(conn, id); err != nil {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Println(cancelStyle.Render("Entry deletion cancelled."))
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