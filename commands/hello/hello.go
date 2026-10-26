// Package hello implements the /hello slash command, which greets the user
// by their server nickname with a randomly chosen greeting.
package hello

import (
	"fmt"
	"math/rand/v2"

	"github.com/bwmarrin/discordgo"
)

// Definition is the slash command registered with Discord.
var Definition = &discordgo.ApplicationCommand{
	Name:        "hello",
	Description: "Get a greeting from the bot",
}

// greetings is the pool of greetings the command picks from.
var greetings = []string{
	"Hello",
	"Howdy",
	"Wassup",
	"Meowdy",
}

// responder is the subset of *discordgo.Session used to reply.
type responder interface {
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
}

// Handle responds to a /hello interaction.
func Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := respond(s, i, rand.IntN); err != nil {
		fmt.Printf("hello: failed to respond: %v\n", err)
	}
}

// respond sends the interaction response using the provided responder and pick function.
func respond(r responder, i *discordgo.InteractionCreate, pick func(n int) int) error {
	return r.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: greeting(displayName(i.Interaction), pick),
		},
	})
}

// greeting builds a greeting for name, using pick to choose from greetings.
func greeting(name string, pick func(n int) int) string {
	return fmt.Sprintf("%s %s!", greetings[pick(len(greetings))], name)
}

// displayName returns the user's server nickname, falling back to their
// global display name and then username.
func displayName(i *discordgo.Interaction) string {
	switch {
	case i.Member != nil && i.Member.User != nil:
		return i.Member.DisplayName()
	case i.User != nil:
		return i.User.DisplayName()
	default:
		return "there"
	}
}
