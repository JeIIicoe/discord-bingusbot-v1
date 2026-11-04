package wordlecount

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// backfillPageLimit is how many pages of 100 messages /wordlebackfill reads at
// most, plenty for two weeks without going through months of a busy channel.
const backfillPageLimit = 20

// channelHistory is the part of *discordgo.Session the backfill uses to read a
// channel and post its progress.
type channelHistory interface {
	messageSender
	ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error)
}

// discardSender posts nothing. It's used when replaying old posts so an old
// week finishing doesn't post its leaderboard.
type discardSender struct{}

func (discardSender) ChannelMessageSendComplex(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
	return nil, nil
}

// backfillSummary collects what a backfill loaded, for the summary at the end.
type backfillSummary struct {
	players         map[string]bool // user IDs of everyone loaded
	lastWeekStart   string
	lastWeekDays    int
	lastWeekSkipped bool // already in the lifetime stats
	thisWeekDays    int
	thisWeekSkipped int // days already in the weekly tally
}

// Backfill handles /wordlebackfill, which sets the bot up in a channel. It
// loads last week's results straight into the lifetime stats and this week's
// into the weekly tally, posting a message for each day as it goes. Anything
// already loaded is skipped, so it's safe to run more than once.
func (tracker *Tracker) Backfill(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if !ownerInServer(session, interaction) {
		return
	}

	// Discord wants an answer within 3 seconds and reading history can take
	// longer, so the reply is deferred and edited at the end.
	err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		fmt.Printf("wordlecount: acknowledging backfill: %v\n", err)
		return
	}

	reply := "Backfill finished."
	err = catch(func() error {
		serverTracker := tracker.forServer(interaction.GuildID)
		return serverTracker.backfill(session, interaction.ChannelID, session.State.User.ID, time.Now())
	})
	if err != nil {
		reply = "⚠️ Backfill failed, see the dump below."
		reportError(session, interaction.ChannelID, err)
	}
	if _, err := session.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{Content: &reply}); err != nil {
		fmt.Printf("wordlecount: replying to backfill: %v\n", err)
	}
}

// backfill loads last week's and this week's results posts in channelID, as
// of now, then posts a summary of who it loaded.
func (tracker *Tracker) backfill(history channelHistory, channelID, botUserID string, now time.Time) error {
	thisWeekStart := WeekStart(PostDay(now, tracker.timezone))
	lastWeekStart := addDays(thisWeekStart, -DaysPerWeek)

	posts, err := tracker.postsSince(history, channelID, botUserID, lastWeekStart)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return say(history, channelID, "I couldn't find any Wordle results from last week or this week, so there's nothing to load yet!")
	}

	var lastWeekPosts, thisWeekPosts []*discordgo.Message
	for _, post := range posts {
		if WeekStart(PostDay(post.Timestamp, tracker.timezone)) == lastWeekStart {
			lastWeekPosts = append(lastWeekPosts, post)
		} else {
			thisWeekPosts = append(thisWeekPosts, post)
		}
	}

	if err := say(history, channelID, "Getting set up! Loading Wordle results from **%s** onwards…", dayLabel(lastWeekStart)); err != nil {
		return err
	}
	summary := &backfillSummary{players: make(map[string]bool), lastWeekStart: lastWeekStart}
	if err := tracker.loadLastWeek(history, channelID, lastWeekStart, lastWeekPosts, summary); err != nil {
		return err
	}
	if err := tracker.loadThisWeek(history, channelID, thisWeekPosts, summary); err != nil {
		return err
	}

	if _, err := history.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{summaryEmbed(summary)},
	}); err != nil {
		return err
	}
	return say(history, channelID, "All caught up! I'll keep score from here!!")
}

// loadLastWeek adds last week's posts straight to the lifetime stats, as that
// week is already over. It skips them if the week is already in there.
func (tracker *Tracker) loadLastWeek(sender messageSender, channelID, weekStart string, posts []*discordgo.Message, summary *backfillSummary) error {
	if len(posts) == 0 {
		return nil
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	var lifetime Lifetime
	if err := loadJSON(tracker.path(lifetimeFile), &lifetime); err != nil {
		return err
	}
	if lifetime.LastWeekMerged >= weekStart {
		summary.lastWeekSkipped = true
		return say(sender, channelID, "Last week's already in the lifetime stats, so I've skipped it")
	}

	week := Week{Start: weekStart}
	for _, post := range posts {
		if !tracker.record(&week, post) {
			continue // the Wordle app posted twice in one day
		}
		summary.lastWeekDays++
		if err := sayDayLoaded(sender, channelID, post, tracker.timezone, summary); err != nil {
			return err
		}
	}

	// same as what would've happened on the Sunday
	week.Finish(&lifetime)
	return saveJSON(tracker.path(lifetimeFile), &lifetime)
}

// loadThisWeek replays this week's posts through process, the same as live
// ones, but with a sender that doesn't post anything.
func (tracker *Tracker) loadThisWeek(sender messageSender, channelID string, posts []*discordgo.Message, summary *backfillSummary) error {
	for _, post := range posts {
		counted, err := tracker.process(discardSender{}, post)
		if err != nil {
			return err
		}
		if !counted {
			summary.thisWeekSkipped++
			date := PostDay(post.Timestamp, tracker.timezone)
			if err := say(sender, channelID, "**%s** was already counted.", dayLabel(date)); err != nil {
				return err
			}
			continue
		}
		summary.thisWeekDays++
		if err := sayDayLoaded(sender, channelID, post, tracker.timezone, summary); err != nil {
			return err
		}
	}
	return nil
}

// sayDayLoaded posts that a day's results have been loaded, naming who played,
// and notes the players in summary.
func sayDayLoaded(sender messageSender, channelID string, post *discordgo.Message, timezone *time.Location, summary *backfillSummary) error {
	displayNames := make(map[string]string, len(post.Mentions))
	for _, user := range post.Mentions {
		displayNames[user.ID] = user.DisplayName()
	}

	var names []string
	for userID := range parseResults(post.Content) {
		summary.players[userID] = true
		if name := displayNames[userID]; name != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	namesText := strings.Join(names, ", ")
	if namesText == "" {
		namesText = "nobody played"
	}

	date := PostDay(post.Timestamp, timezone)
	return say(sender, channelID, "Loaded **%s** with %s", dayLabel(date), namesText)
}

// summaryEmbed lists who a backfill loaded, and what it loaded for each week.
func summaryEmbed(summary *backfillSummary) *discordgo.MessageEmbed {
	userIDs := make([]string, 0, len(summary.players))
	for userID := range summary.players {
		userIDs = append(userIDs, userID)
	}
	slices.Sort(userIDs)

	mentions := make([]string, len(userIDs))
	for index, userID := range userIDs {
		mentions[index] = "<@" + userID + ">"
	}
	playersText := strings.Join(mentions, "\n")
	if playersText == "" {
		playersText = "Nobody new"
	}

	lastWeekStart, _ := time.Parse(dateLayout, summary.lastWeekStart)
	lastWeekEnd := lastWeekStart.AddDate(0, 0, DaysPerWeek-1)
	lastWeekText := fmt.Sprintf("%s – %s\n", lastWeekStart.Format("2 Jan"), lastWeekEnd.Format("2 Jan"))
	switch {
	case summary.lastWeekSkipped:
		lastWeekText += "Already in the lifetime stats"
	case summary.lastWeekDays == 0:
		lastWeekText += "No results found"
	default:
		lastWeekText += fmt.Sprintf("%d day(s), added to the lifetime stats", summary.lastWeekDays)
	}

	thisWeekText := fmt.Sprintf("%d day(s) loaded", summary.thisWeekDays)
	if summary.thisWeekSkipped > 0 {
		thisWeekText += fmt.Sprintf("\n%d already counted", summary.thisWeekSkipped)
	}

	return &discordgo.MessageEmbed{
		Title:       "Loaded players",
		Description: playersText,
		Color:       tallyColour,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Last week", Value: lastWeekText, Inline: true},
			{Name: "This week so far", Value: thisWeekText, Inline: true},
		},
	}
}

// say posts a plain message to channelID. Mentions in it don't ping anyone.
func say(sender messageSender, channelID, format string, args ...any) error {
	_, err := sender.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         fmt.Sprintf(format, args...),
		AllowedMentions: &discordgo.MessageAllowedMentions{},
	})
	return err
}

// dayLabel formats a YYYY-MM-DD date as, say, "Monday 26 Oct".
func dayLabel(date string) string {
	parsedDate, _ := time.Parse(dateLayout, date)
	return parsedDate.Format("Monday 2 Jan")
}

// addDays returns the YYYY-MM-DD date days after date.
func addDays(date string, days int) string {
	parsedDate, _ := time.Parse(dateLayout, date)
	return parsedDate.AddDate(0, 0, days).Format(dateLayout)
}

// postsSince returns channelID's Wordle results posts from date onwards,
// oldest first, reading back at most backfillPageLimit pages of history.
func (tracker *Tracker) postsSince(history channelHistory, channelID, botUserID, date string) ([]*discordgo.Message, error) {
	var posts []*discordgo.Message
	beforeID := ""
	for range backfillPageLimit {
		page, err := history.ChannelMessages(channelID, 100, beforeID, "", "")
		if err != nil {
			return nil, fmt.Errorf("reading channel history: %w", err)
		}

		// Messages come newest first, so the first one before date means
		// there's nothing left worth reading.
		for _, message := range page {
			if PostDay(message.Timestamp, tracker.timezone) < date {
				slices.Reverse(posts)
				return posts, nil
			}
			if isWordlePost(message, botUserID) {
				posts = append(posts, message)
			}
		}
		// a short page means it's hit the begining of the channel
		if len(page) < 100 {
			break
		}
		beforeID = page[len(page)-1].ID
	}

	slices.Reverse(posts) // oldest first
	return posts, nil
}
