// Package wordlecount watches for the Wordle app's daily results post, keeps
// a weekly tally of everyone's scores, and posts a leaderboard at the end of
// each Monday–Sunday week.
//
// Each game scores its number of guesses; a failed game or a missed day
// scores MaxScore. Lowest weekly score wins. The running week is kept in
// weekly_scores.json and, once the leaderboard is posted, merged into
// lifetime_tally.json and wiped.
package wordlecount

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	tallyColour       = 0x6AAA64 // Wordle green
	leaderboardColour = 0xC9B458 // Wordle yellow
)

// messageSender is the part of *discordgo.Session used to post messages, so it
// can be faked in tests.
type messageSender interface {
	ChannelMessageSendComplex(channelID string, message *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// Tracker tallies Wordle results, storing its files in a data directory.
// Each server gets its own subdirectory, see forServer.
type Tracker struct {
	mutex    *sync.Mutex // shared by every server's tracker, see forServer
	dataDir  string
	timezone *time.Location
}

// New returns a Tracker that stores its files in dataDir and decides which
// day a results post belongs to using timezone.
func New(dataDir string, timezone *time.Location) *Tracker {
	return &Tracker{mutex: new(sync.Mutex), dataDir: dataDir, timezone: timezone}
}

// forServer returns a copy of the tracker pointed at one server's folder. The
// lock is shared accross every server, which is fine for how little traffic
// the bot gets.
func (tracker *Tracker) forServer(guildID string) *Tracker {
	serverTracker := *tracker
	serverTracker.dataDir = filepath.Join(tracker.dataDir, guildID)
	return &serverTracker
}

// Handle is a MessageCreate handler that processes Wordle results posts.
// Errors and panics are posted to the channel as a dump.
func (tracker *Tracker) Handle(session *discordgo.Session, event *discordgo.MessageCreate) {
	// DMs are ignored, scores are only kept per server.
	if event.GuildID == "" || !isWordlePost(event.Message, session.State.User.ID) {
		return
	}
	err := catch(func() error {
		counted, err := tracker.forServer(event.GuildID).process(session, event.Message)
		if err != nil || !counted {
			return err
		}
		_ = session.MessageReactionAdd(event.ChannelID, event.ID, "👀")
		return nil
	})
	if err != nil {
		reportError(session, event.ChannelID, err)
	}
}

// isWordlePost reports whether message is a results post from the Wordle app,
// and not something the bot itself posted.
func isWordlePost(message *discordgo.Message, botUserID string) bool {
	return message.Author != nil &&
		message.Author.Bot &&
		message.Author.ID != botUserID &&
		isResultsPost(message.Content)
}

// process updates the weekly tally with the results from a single post. It
// reports whether the post was counted, which it isn't if its day already
// has been or its week has already finished.
func (tracker *Tracker) process(sender messageSender, message *discordgo.Message) (bool, error) {
	date := PostDay(message.Timestamp, tracker.timezone)
	weekStart := WeekStart(date)

	// Without the lock, two posts at once could both read the file and the
	// second save would wipe out the first.
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	var week Week
	if err := loadJSON(tracker.path(weeklyFile), &week); err != nil {
		return false, err
	}

	if week.Start != "" && week.Start != weekStart {
		if weekStart < week.Start {
			return false, nil // a post from a week that's already finished
		}
		// Sunday's post never arrived, so close out the old week first.
		if err := tracker.finish(sender, &week, nil); err != nil {
			return false, err
		}
		week = Week{}
	}
	if !tracker.record(&week, message) {
		return false, nil // already counted this day
	}

	// Sunday finishes the week off with the leaderboard.
	if week.DayIndex(date) == DaysPerWeek-1 {
		return true, tracker.finish(sender, &week, message)
	}
	return true, saveJSON(tracker.path(weeklyFile), &week)
}

// record adds the results from message to week, starting the week if it's
// empty. It returns false if the message's day has already been counted.
func (tracker *Tracker) record(week *Week, message *discordgo.Message) bool {
	displayNames := make(map[string]string, len(message.Mentions))
	for _, user := range message.Mentions {
		displayNames[user.ID] = user.DisplayName()
	}
	date := PostDay(message.Timestamp, tracker.timezone)

	if week.Start == "" {
		week.Start = WeekStart(date)
	}
	week.ChannelID = message.ChannelID
	return week.Record(date, parseResults(message.Content), displayNames)
}

// finish posts the week's leaderboard, adds the week to the lifetime tally
// and wipes the weekly file. The leaderboard replies to replyTo if set.
func (tracker *Tracker) finish(sender messageSender, week *Week, replyTo *discordgo.Message) error {
	var lifetime Lifetime
	if err := loadJSON(tracker.path(lifetimeFile), &lifetime); err != nil {
		return err
	}
	merged := week.Finish(&lifetime)

	leaderboard := &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{leaderboardEmbed(week)}}
	if replyTo != nil {
		leaderboard.Reference = replyTo.Reference()
	}
	// Still save the week if posting fails so the scores aren't lost.
	_, postErr := sender.ChannelMessageSendComplex(week.ChannelID, leaderboard)
	if postErr != nil {
		postErr = fmt.Errorf("posting leaderboard: %w", postErr)
	}

	// Lifetime is saved before the weekly file goes. If the delete fails, the
	// week gets finished again next time and LastWeekMerged stops it counting twice.
	if merged {
		if err := saveJSON(tracker.path(lifetimeFile), &lifetime); err != nil {
			return errors.Join(postErr, err)
		}
	}
	if err := os.Remove(tracker.path(weeklyFile)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return errors.Join(postErr, err)
	}
	return postErr
}

// path returns where fileName is stored in the tracker's data directory.
func (tracker *Tracker) path(fileName string) string {
	return filepath.Join(tracker.dataDir, fileName)
}

// leaderboardEmbed shows the week's final standings.
func leaderboardEmbed(week *Week) *discordgo.MessageEmbed {
	weekStart, _ := time.Parse(dateLayout, week.Start)
	weekEnd := weekStart.AddDate(0, 0, DaysPerWeek-1)
	// Embed titles can't be made bigger, so the title is a heading in the
	// description instead.
	heading := fmt.Sprintf("# 🏆 Wordle Weekly Leaderboard\n**%s – %s**\n\n", weekStart.Format("2 Jan"), weekEnd.Format("2 Jan"))
	return &discordgo.MessageEmbed{
		Description: heading + standingsText(week),
		Color:       leaderboardColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: ("See you next week! <3 from Pancake"),
		},
	}
}

// medals are shown in place of the top three ranks.
var medals = []string{"🥇", "🥈", "🥉"}

// standingsText lists the week's players by rank. Ties share a line, and the
// top three get medals and a bigger line.
func standingsText(week *Week) string {
	standings := week.Standings()
	if len(standings) == 0 {
		return "Dead game" // Needed to handle gracefully
	}

	var text strings.Builder
	for _, tied := range groupByRank(standings) {
		rank := tied[0].Rank
		if rank <= len(medals) {
			fmt.Fprintf(&text, "### %s %s\n", medals[rank-1], tiedPlayersText(tied))
			continue
		}
		if rank == len(medals)+1 {
			text.WriteString("\n") // gap between the medals and everyone else
		}
		fmt.Fprintf(&text, "**%s** %s\n", ordinal(rank), tiedPlayersText(tied))
	}
	return text.String()
}

// groupByRank splits standings into runs of players who share a rank.
func groupByRank(standings []Standing) [][]Standing {
	var groups [][]Standing
	groupStart := 0
	for position := range standings {
		if position > 0 && standings[position].Rank != standings[groupStart].Rank {
			groups = append(groups, standings[groupStart:position])
			groupStart = position
		}
	}
	return append(groups, standings[groupStart:])
}

// tiedPlayersText names players who share a rank, followed by their score.
// Days played goes at the end if they all match, otherwise next to each name.
func tiedPlayersText(tied []Standing) string {
	sameDaysPlayed := true
	for _, standing := range tied {
		if standing.DaysPlayed != tied[0].DaysPlayed {
			sameDaysPlayed = false
		}
	}

	names := make([]string, len(tied))
	for index, standing := range tied {
		name := "<@" + standing.UserID + ">"
		if !sameDaysPlayed {
			name += fmt.Sprintf(" (%d/%d)", standing.DaysPlayed, DaysPerWeek)
		}
		if standing.DaysPlayed == DaysPerWeek {
			name += " 🔥"
		}
		names[index] = name
	}

	line := fmt.Sprintf("%s — **%d**/%d", strings.Join(names, ", "), tied[0].Score, DaysPerWeek*MaxScore)
	if sameDaysPlayed {
		line += fmt.Sprintf(" + %d/%d days", tied[0].DaysPlayed, DaysPerWeek)
	}
	return line
}

// ordinal turns a rank into 4th, 11th, 22nd and so on.
func ordinal(rank int) string {
	suffix := "th"
	switch rank % 10 {
	case 1:
		suffix = "st"
	case 2:
		suffix = "nd"
	case 3:
		suffix = "rd"
	}
	if teens := rank % 100; teens >= 11 && teens <= 13 {
		suffix = "th"
	}
	return fmt.Sprintf("%d%s", rank, suffix)
}
