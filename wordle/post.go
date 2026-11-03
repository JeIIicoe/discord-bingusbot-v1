package wordlecount

import (
	"github.com/bwmarrin/discordgo"
)

// PostLeaderboard handles /post: it posts this week's leaderboard in the
// channel straight away, scored as if the week had ended. The tally itself is
// left alone, so the real leaderboard still posts on Sunday as usual.
func (tracker *Tracker) PostLeaderboard(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if !ownerInServer(session, interaction) {
		return
	}

	var summary string
	err := catch(func() error {
		var err error
		summary, err = tracker.forServer(interaction.GuildID).postLeaderboard(session, interaction.ChannelID)
		return err
	})
	if err != nil {
		summary = "⚠️ Posting the leaderboard failed, see the dump below."
		reportError(session, interaction.ChannelID, err)
	}
	respond(session, interaction, summary)
}

// postLeaderboard posts the current week's leaderboard to channelID. Days
// still to come count as missed, as they would if the week ended now, and
// nothing is saved.
func (tracker *Tracker) postLeaderboard(sender messageSender, channelID string) (string, error) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	var week Week
	if err := loadJSON(tracker.path(weeklyFile), &week); err != nil {
		return "", err
	}
	if week.Start == "" {
		return "No results recorded this week yet.", nil
	}

	week.backdate(DaysPerWeek)
	_, err := sender.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{leaderboardEmbed(&week)},
	})
	if err != nil {
		return "", err
	}
	return "Posted this week's leaderboard.", nil
}
