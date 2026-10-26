package wordlecount

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// watch for posted message

// handles posting

// does stuff
func process(message *discordgo.Message) error {
	results := parseResults(message.Content)
	playerNames := (message.Mentions) // Do I need to go through string map? todo investigate

	// get usernames from player mentions
	for _, user := range message.Mentions {
		nicknames[user.ID] = user.DisplayName()
	}

	var week Week
	// load week data loadJson()

	// load data into var struct
	// type Week struct {
	// Start     string                 `json:"weekStart"`     // Monday, YYYY-MM-DD ? cant do M T W etc
	// ChannelID string                 `json:"channelId"`     // good to future proof incase other servers use
	// Days      []string               `json:"processedDays"` // keep track of which days have been processed?
	// Players   map[string]*WeekPlayer `json:"players"`
	// }

	// get date
	if !week.Record(date, results, nicknames) {
		return fmt.Errorf("failed to record results for date %v", date)
	}
	return nil
}
