// Package handler wires slash commands from the commands/ folder into the bot.
package handler

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"bingusbot/commands/hello"
)

// commandHandler is the function that answers a slash command.
type commandHandler = func(session *discordgo.Session, interaction *discordgo.InteractionCreate)

// Command pairs a slash command definition with the function that handles it.
type Command struct {
	definition *discordgo.ApplicationCommand
	handler    commandHandler
}

// Commands lists every slash command the bot exposes. To add a command,
// create a package under commands/ exporting definition and handle, then add
// it here. Commands that need state set up in main, like the Wordle
// commands, are built with NewCommand and added there instead.
var Commands = []Command{
	{definition: hello.Definition, handler: hello.Handle},
}

// NewCommand pairs a slash command definition with its handler.
func NewCommand(definition *discordgo.ApplicationCommand, handler commandHandler) Command {
	return Command{definition: definition, handler: handler}
}

// commandRegistrar is the subset of *discordgo.Session LoadCommands needs, so
// tests can substitute a fake.
type commandRegistrar interface {
	AddHandler(handler interface{}) func()
	ApplicationCommandBulkOverwrite(appID string, guildID string, commands []*discordgo.ApplicationCommand, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
}

// LoadCommands registers commands with Discord, in every server the bot is in,
// and routes incoming interactions to their handlers. Registration overwrites
// the application's existing commands, so commands removed from the list are
// also removed from Discord.
//
// appID is the bot's application ID, available as session.State.User.ID once
// the session is open.
func LoadCommands(registrar commandRegistrar, appID string, commands []Command) error {
	handlers, err := buildHandlers(commands)
	if err != nil {
		return err
	}

	definitions := make([]*discordgo.ApplicationCommand, len(commands))
	for index, command := range commands {
		definitions[index] = command.definition
	}

	// An empty guild ID registers the commands globally.
	if _, err := registrar.ApplicationCommandBulkOverwrite(appID, "", definitions); err != nil {
		return fmt.Errorf("registering commands: %w", err)
	}

	registrar.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		route(handlers, session, interaction)
	})

	return nil
}

// buildHandlers constructs a map from command names to their handlers, ensuring no duplicates or missing definitions.
func buildHandlers(commands []Command) (map[string]commandHandler, error) {
	handlers := make(map[string]commandHandler, len(commands))
	for _, command := range commands {

		if command.definition == nil || command.definition.Name == "" {
			return nil, fmt.Errorf("command has no definition or name")
		}

		if command.handler == nil {
			return nil, fmt.Errorf("command %q has no handler", command.definition.Name)
		}

		if _, exists := handlers[command.definition.Name]; exists {
			return nil, fmt.Errorf("duplicate command name %q", command.definition.Name)
		}

		handlers[command.definition.Name] = command.handler
	}

	return handlers, nil
}

// route dispatches an interaction to the appropriate command handler based on its name.
func route(handlers map[string]commandHandler, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if handle, found := handlers[interaction.ApplicationCommandData().Name]; found {
		handle(session, interaction)
	}
}
