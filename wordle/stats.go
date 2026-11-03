package wordlecount

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Stats handles /stats: it shows the invoker their lifetime Wordle stats for
// this server, including this week so far. Anyone can use it.
func (tracker *Tracker) Stats(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	member := interaction.Member
	if interaction.GuildID == "" || member == nil || member.User == nil {
		respond(session, interaction, "Use this command in a server.")
		return
	}

	var stats LifetimePlayer
	var found bool
	err := catch(func() error {
		var err error
		stats, found, err = tracker.forServer(interaction.GuildID).lifetimeStats(member.User.ID)
		return err
	})
	switch {
	case err != nil:
		respond(session, interaction, "⚠️ I couldn't load your stats, see the dump below.")
		reportError(session, interaction.ChannelID, err)
		return
	case !found:
		respond(session, interaction, "You're not in there yet silly! Play a Wordle with the group and you'll show up here!")
		return
	}

	err = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{statsEmbed(member, stats)},
		},
	})
	if err != nil {
		fmt.Printf("wordlecount: responding to stats: %v\n", err)
	}
}

// lifetimeStats returns userID's lifetime stats with this week so far added
// on, since a week only joins the lifetime stats once it's over. found is
// false if they've never played.
func (tracker *Tracker) lifetimeStats(userID string) (LifetimePlayer, bool, error) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	var lifetime Lifetime
	if err := loadJSON(tracker.path(lifetimeFile), &lifetime); err != nil {
		return LifetimePlayer{}, false, err
	}
	var week Week
	if err := loadJSON(tracker.path(weeklyFile), &week); err != nil {
		return LifetimePlayer{}, false, err
	}

	var stats LifetimePlayer
	lifetimePlayer, found := lifetime.Players[userID]
	if found {
		stats = *lifetimePlayer
	}

	// The weekly tally is only added if it hasn't been merged in already,
	// so no game is counted twice.
	weekPlayer, playingThisWeek := week.Players[userID]
	if playingThisWeek && week.Start > lifetime.LastWeekMerged {
		stats.DaysPlayed += len(weekPlayer.Results)
		stats.TotalScore += sum(weekPlayer.Results)
		stats.WeeksPlayed++
		found = true
	}

	return stats, found, nil
}

// statsEmbed shows a player's lifetime stats as a card titled with their
// server name, with their avatar alongside.
func statsEmbed(member *discordgo.Member, stats LifetimePlayer) *discordgo.MessageEmbed {
	average := 0.0
	if stats.DaysPlayed > 0 {
		average = float64(stats.TotalScore) / float64(stats.DaysPlayed)
	}

	return &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("%s's lifetime Wordle stats!", member.DisplayName()),
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: member.User.AvatarURL("128")},
		Color:     tallyColour,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Average", Value: fmt.Sprintf("**%.2f** guesses", average), Inline: true},
			{Name: "Games played", Value: fmt.Sprintf("**%d**", stats.DaysPlayed), Inline: true},
			{Name: "Weeks played", Value: fmt.Sprintf("**%d**", stats.WeeksPlayed), Inline: true},
			{Name: "Weeks won", Value: fmt.Sprintf("**%d**", stats.WeeksWon), Inline: true},
			{Name: "Full weeks", Value: fmt.Sprintf("**%d**", stats.CompleteWeeks), Inline: true},
			{Name: "Total guesses", Value: fmt.Sprintf("**%d**", stats.TotalScore), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("This also includes this week so far!"),
		},
	}
}
