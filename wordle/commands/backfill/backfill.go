// Package backfill implements the /wordlebackfill slash command, which sets the
// bot up in a server by loading last week's and this week's results.
package backfill

import (
	"github.com/bwmarrin/discordgo"

	wordlecount "bingusbot/wordle"
)

// Definition is the slash command registered with Discord. Only the bot owner
// can use it; the handler checks this.
var Definition = &discordgo.ApplicationCommand{
	Name:        "wordlebackfill",
	Description: "Catch the Wordle tally up on this week's results posts in this channel",
}

// Handle returns the /wordlebackfill handler for tracker.
func Handle(tracker *wordlecount.Tracker) func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	return tracker.Backfill
}
