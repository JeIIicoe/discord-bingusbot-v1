// Package post implements the /post slash command, which posts this week's
// Wordle leaderboard straight away, scored as if the week had ended.
package post

import (
	"github.com/bwmarrin/discordgo"

	wordlecount "bingusbot/wordle"
)

// Definition is the slash command registered with Discord. Only the bot owner
// can use it; the handler checks this.
var Definition = &discordgo.ApplicationCommand{
	Name:        "post",
	Description: "Post this week's Wordle leaderboard now, as if the week had ended",
}

// Handle returns the /post handler for tracker.
func Handle(tracker *wordlecount.Tracker) func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	return tracker.PostLeaderboard
}
