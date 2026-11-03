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

// Commands is the list of slash commands that don't need anything set up
// first. The Wordle ones need the tracker, so main adds those with NewCommand.
var Commands = []Command{
	{definition: hello.Definition, handler: hello.Handle},
}

// NewCommand pairs a slash command definition with its handler.
func NewCommand(definition *discordgo.ApplicationCommand, handler commandHandler) Command {
	return Command{definition: definition, handler: handler}
}

// commandRegistrar is the part of *discordgo.Session LoadCommands uses, so it
// can be faked in tests.
type commandRegistrar interface {
	AddHandler(handler interface{}) func()
	ApplicationCommandBulkOverwrite(appID string, guildID string, commands []*discordgo.ApplicationCommand, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
}

// LoadCommands registers commands with Discord, in every server the bot is in,
// and routes interactions to their handlers. It replaces whatever was
// registered before, so a command taken out of the list disappears from
// Discord too. appID is the bot's user ID.
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

// buildHandlers maps each command name to its handler, and errors on a
// missing definition or a name used twice.
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

// route passes a slash command interaction to the handler with its name.
func route(handlers map[string]commandHandler, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if handle, found := handlers[interaction.ApplicationCommandData().Name]; found {
		handle(session, interaction)
	}
}
