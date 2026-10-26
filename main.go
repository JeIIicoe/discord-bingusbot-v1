package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"bingusbot/handler"
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
	guildID := os.Getenv("DISCORD_GUILD_ID")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsGuilds

	if err := s.Open(); err != nil {
		return fmt.Errorf("opening connection: %w", err)
	}
	defer s.Close()

	if err := handler.LoadCommands(s, s.State.User.ID, guildID, handler.Commands); err != nil {
		return err
	}
	fmt.Printf("Logged in as %s with %d command(s). Press Ctrl+C to exit.\n", s.State.User.Username, len(handler.Commands))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
	return nil
}
