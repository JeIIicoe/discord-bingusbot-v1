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

// poster is the subset of *discordgo.Session used to post messages, so tests
// can substitute a fake.
type poster interface {
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// Tracker tallies Wordle results, storing its files in a data directory.
type Tracker struct {
	mu  sync.Mutex
	dir string
	loc *time.Location
}

// New returns a Tracker that stores its files in dir and decides which day a
// results post belongs to using loc.
func New(dir string, loc *time.Location) *Tracker {
	return &Tracker{dir: dir, loc: loc}
}

// Handle is a MessageCreate handler that processes Wordle results posts.
func (t *Tracker) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || !m.Author.Bot || m.Author.ID == s.State.User.ID || !isResultsPost(m.Content) {
		return
	}
	if err := t.process(s, m.Message); err != nil {
		fmt.Printf("wordlecount: %v\n", err)
	}
}

// process updates the weekly tally with the results from a single post.
// It returns an error if reading or writing the tally fails.
func (t *Tracker) process(p poster, m *discordgo.Message) error {
	results := parseResults(m.Content)
	names := make(map[string]string, len(m.Mentions))
	for _, u := range m.Mentions {
		names[u.ID] = u.DisplayName()
	}
	date := PostDay(m.Timestamp, t.loc)
	start := WeekStart(date)

	t.mu.Lock()
	defer t.mu.Unlock()

	var week Week
	if err := loadJSON(t.path(weeklyFile), &week); err != nil {
		return err
	}
	if week.Start != "" && week.Start != start {
		if start < week.Start {
			return nil // a post from a week that's already finished
		}
		// Sunday's post never arrived, so close out the old week first.
		if err := t.finish(p, &week, nil); err != nil {
			return err
		}
		week = Week{}
	}
	if week.Start == "" {
		week.Start = start
	}
	week.ChannelID = m.ChannelID
	if !week.Record(date, results, names) {
		return nil // already counted this day
	}

	if week.DayIndex(date) == DaysPerWeek-1 {
		return t.finish(p, &week, m)
	}
	if err := saveJSON(t.path(weeklyFile), &week); err != nil {
		return err
	}
	_, err := p.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:    []*discordgo.MessageEmbed{dailyEmbed(&week, date)},
		Reference: m.Reference(),
	})
	return err
}

// finish posts the week's leaderboard, adds the week to the lifetime tally
// and wipes the weekly file. The leaderboard replies to replyTo if set.
func (t *Tracker) finish(p poster, week *Week, replyTo *discordgo.Message) error {
	var lifetime Lifetime
	if err := loadJSON(t.path(lifetimeFile), &lifetime); err != nil {
		return err
	}
	merged := week.Finish(&lifetime)

	msg := &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{leaderboardEmbed(week)}}
	if replyTo != nil {
		msg.Reference = replyTo.Reference()
	}
	if _, err := p.ChannelMessageSendComplex(week.ChannelID, msg); err != nil {
		// Still save the week so the scores aren't lost.
		fmt.Printf("wordlecount: posting leaderboard: %v\n", err)
	}

	if merged {
		if err := saveJSON(t.path(lifetimeFile), &lifetime); err != nil {
			return err
		}
	}
	if err := os.Remove(t.path(weeklyFile)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (t *Tracker) path(name string) string {
	return filepath.Join(t.dir, name)
}

func dailyEmbed(week *Week, date string) *discordgo.MessageEmbed {
	d, _ := time.Parse(dateLayout, date)
	day := week.DayIndex(date) + 1
	return &discordgo.MessageEmbed{
		Title:       "📊 Wordle Weekly Tally — " + d.Format("Monday 2 Jan"),
		Description: standingsText(week, day*MaxScore, day),
		Color:       tallyColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Day %d of %d · Lowest score wins · Missed days count as %d", day, DaysPerWeek, MaxScore),
		},
	}
}

func leaderboardEmbed(week *Week) *discordgo.MessageEmbed {
	start, _ := time.Parse(dateLayout, week.Start)
	end := start.AddDate(0, 0, DaysPerWeek-1)
	return &discordgo.MessageEmbed{
		Title:       "🏆 Wordle Weekly Leaderboard",
		Description: fmt.Sprintf("**%s – %s**\n\n", start.Format("2 Jan"), end.Format("2 Jan")) + standingsText(week, DaysPerWeek*MaxScore, DaysPerWeek),
		Color:       leaderboardColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Lowest score wins · Missed days count as %d · See you next week!", MaxScore),
		},
	}
}

// standingsText lists the week's players, one line each, scored out of
// maxScore over the given number of days.
func standingsText(week *Week, maxScore, days int) string {
	standings := week.Standings()
	if len(standings) == 0 {
		return "Dead game" // Needed to handle gracefully
	}
	var b strings.Builder
	for _, s := range standings {
		fmt.Fprintf(&b, "%s <@%s> — **%d**/%d · %d/%d days", rankLabel(s.Rank), s.UserID, s.Score, maxScore, s.DaysPlayed, days)
		if s.DaysPlayed == DaysPerWeek {
			b.WriteString(" 🔥")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func rankLabel(rank int) string {
	switch rank {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("`%d.`", rank)
	}
}
