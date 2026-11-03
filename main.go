package main

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"
	_ "time/tzdata" // embed timezone data so WORDLE_TIMEZONE works on any OS

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"bingusbot/handler"
	wordlecount "bingusbot/wordle"
	"bingusbot/wordle/commands/backfill"
	"bingusbot/wordle/commands/post"
	"bingusbot/wordle/commands/stats"
)

// main is the entry point of the application. It initializes and runs the Discord bot.
func main() {
	fmt.Printf("Loading .env file...\n")
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "error loading .env file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf(".env file loaded successfully.\n")

	fmt.Printf("Starting bot...\n")
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run initializes and starts the Discord bot, handling connection and command registration.
// It blocks until an interrupt signal is received.
func run() error {
	fmt.Println("Discord bot starting...")

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return fmt.Errorf("DISCORD_TOKEN is not set")
	}
	dataDir := envOr("DATA_DIR", "data")
	timezone, err := time.LoadLocation(envOr("WORDLE_TIMEZONE", "Europe/London"))
	if err != nil {
		return fmt.Errorf("loading WORDLE_TIMEZONE: %w", err)
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	// Message content is a privileged intent: enable it under Bot in the
	// Discord developer portal or the Wordle tracker sees empty messages.
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	tracker := wordlecount.New(dataDir, timezone)
	session.AddHandler(tracker.Handle)

	commands := slices.Concat(handler.Commands, []handler.Command{
		handler.NewCommand(backfill.Definition, backfill.Handle(tracker)),
		handler.NewCommand(post.Definition, post.Handle(tracker)),
		handler.NewCommand(stats.Definition, stats.Handle(tracker)),
	})

	if err := session.Open(); err != nil {
		return fmt.Errorf("opening connection: %w", err)
	}
	defer session.Close()

	if err := handler.LoadCommands(session, session.State.User.ID, commands); err != nil {
		return err
	}
	fmt.Printf("Logged in as %s with %d command(s). Press Ctrl+C to exit.\n", session.State.User.Username, len(commands))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
	return nil
}

// envOr returns the environment variable key, or fallback if it isn't set.
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
