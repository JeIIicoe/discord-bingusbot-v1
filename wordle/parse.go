package wordlecount

import (
	"regexp"
	"strconv"
)

const (
	// MaxScore is the worst score for a single day. A failed game (X/6) and a
	// missed day both count as MaxScore.
	MaxScore = 7
	// DaysPerWeek is the number of games in a week.
	DaysPerWeek = 7
)

var (
	// The Wordle app opens its daily results post with this line.
	streakRe     = regexp.MustCompile(`(?i)your group is on an? \d+ day streak`)
	resultLineRe = regexp.MustCompile(`(?m)([1-6X])/6:(.*)$`)
	mentionRe    = regexp.MustCompile(`<@!?(\d+)>`)
)

// IsResultsPost reports whether content is the Wordle app's daily results post.
func isResultsPost(content string) bool {
	return streakRe.MatchString(content)
}

// ParseResults extracts each mentioned player's score from a results post,
// keyed by user ID. Failed games score MaxScore.
func parseResults(content string) map[string]int {
	results := make(map[string]int)
	for _, resultLine := range resultLineRe.FindAllStringSubmatch(content, -1) {
		score := MaxScore

		if resultLine[1] != "X" {
			score, _ = strconv.Atoi(resultLine[1])
		}

		for _, mention := range mentionRe.FindAllStringSubmatch(resultLine[2], -1) {
			results[mention[1]] = score
		}
	}

	return results
}
