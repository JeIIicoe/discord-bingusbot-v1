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
	// Oki wanted the following
	"Hallo",
	"Ahoy",
	"Konnichiwa",
	"Bonjour",
	"Morn",
}

// interactionResponder is the part of *discordgo.Session used to reply.
type interactionResponder interface {
	InteractionRespond(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
}

// Handle responds to a /hello interaction.
func Handle(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if err := respond(session, interaction, rand.IntN); err != nil {
		fmt.Printf("hello: failed to respond: %v\n", err)
	}
}

// respond replies with a random greeting, picked by pick.
func respond(responder interactionResponder, interaction *discordgo.InteractionCreate, pick func(n int) int) error {
	return responder.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: greeting(displayName(interaction.Interaction), pick),
		},
	})
}

// greeting builds a greeting for name, using pick to choose from greetings.
func greeting(name string, pick func(n int) int) string {
	return fmt.Sprintf("%s %s!", greetings[pick(len(greetings))], name)
}

// displayName returns the user's server nickname, falling back to their
// global display name and then username.
func displayName(interaction *discordgo.Interaction) string {
	switch {
	case interaction.Member != nil && interaction.Member.User != nil:
		return interaction.Member.DisplayName()
	case interaction.User != nil:
		return interaction.User.DisplayName()
	default:
		return "there"
	}
}
