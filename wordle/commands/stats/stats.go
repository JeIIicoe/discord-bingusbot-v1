// Package stats implements the /stats slash command, which shows the invoker
// their lifetime Wordle stats for the server.
package stats

import (
	"github.com/bwmarrin/discordgo"

	wordlecount "bingusbot/wordle"
)

// Definition is the slash command registered with Discord. Anyone can use it.
var Definition = &discordgo.ApplicationCommand{
	Name:        "stats",
	Description: "See your lifetime Wordle stats for this server",
}

// Handle returns the /stats handler for tracker.
func Handle(tracker *wordlecount.Tracker) func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	return tracker.Stats
}
