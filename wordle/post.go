package wordlecount

import (
	"github.com/bwmarrin/discordgo"
)

// PostLeaderboard handles /post, which posts this week's leaderboard early as
// if the week had ended. The tally isn't touched, so Sunday still works as normal.
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

// postLeaderboard posts the current week's leaderboard to channelID. Days that
// haven't happened yet count as missed, and nothing gets saved.
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
