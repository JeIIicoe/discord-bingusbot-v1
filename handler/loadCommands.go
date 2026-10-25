// Package handler wires slash commands from the commands/ folder into the bot.
package handler

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"bingusbot/commands/hello"
)

// Command pairs a slash command definition with the function that handles it.
type Command struct {
	Definition *discordgo.ApplicationCommand
	Handler    func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// Commands lists every slash command the bot exposes. To add a command,
// create a package under commands/ exporting Definition and Handle, then add
// it here.
var Commands = []Command{
	{Definition: hello.Definition, Handler: hello.Handle},
}

// session is the subset of *discordgo.Session LoadCommands needs, so tests
// can substitute a fake.
type session interface {
	AddHandler(handler interface{}) func()
	ApplicationCommandBulkOverwrite(appID string, guildID string, commands []*discordgo.ApplicationCommand, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
}

// LoadCommands registers cmds with Discord and routes incoming interactions to
// their handlers. Registration overwrites the application's existing commands,
// so commands removed from cmds are also removed from Discord. An empty
// guildID registers commands globally; a guild ID registers them for that
// guild only, which takes effect immediately and suits development.
//
// appID is the bot's application ID, available as s.State.User.ID once the
// session is open.
func LoadCommands(s session, appID, guildID string, cmds []Command) error {
	handlers, err := buildHandlers(cmds)
	if err != nil {
		return err
	}

	defs := make([]*discordgo.ApplicationCommand, len(cmds))
	for i, c := range cmds {
		defs[i] = c.Definition
	}
	if _, err := s.ApplicationCommandBulkOverwrite(appID, guildID, defs); err != nil {
		return fmt.Errorf("registering commands: %w", err)
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		route(handlers, s, i)
	})
	return nil
}

func buildHandlers(cmds []Command) (map[string]func(*discordgo.Session, *discordgo.InteractionCreate), error) {
	handlers := make(map[string]func(*discordgo.Session, *discordgo.InteractionCreate), len(cmds))
	for _, c := range cmds {
		if c.Definition == nil || c.Definition.Name == "" {
			return nil, fmt.Errorf("command has no definition or name")
		}
		if c.Handler == nil {
			return nil, fmt.Errorf("command %q has no handler", c.Definition.Name)
		}
		if _, dup := handlers[c.Definition.Name]; dup {
			return nil, fmt.Errorf("duplicate command name %q", c.Definition.Name)
		}
		handlers[c.Definition.Name] = c.Handler
	}
	return handlers, nil
}

// route dispatches an interaction to the appropriate command handler based on its name.
func route(handlers map[string]func(*discordgo.Session, *discordgo.InteractionCreate), s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
		h(s, i)
	}
}
