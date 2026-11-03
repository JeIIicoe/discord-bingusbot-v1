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
	streakPattern = regexp.MustCompile(`(?i)your group is on an? \d+ day streak`)

	// e.g. "👑 3/6: @someone @someone-else", everyone on the line got that score
	resultLinePattern = regexp.MustCompile(`(?m)([1-6X])/6:(.*)$`)

	// <@123>, or <@!123> from older clients
	mentionPattern = regexp.MustCompile(`<@!?(\d+)>`)
)

// isResultsPost reports whether content is the Wordle app's daily results post.
func isResultsPost(content string) bool {
	return streakPattern.MatchString(content)
}

// parseResults extracts each mentioned player's score from a results post,
// keyed by user ID. Failed games score MaxScore.
func parseResults(content string) map[string]int {
	scores := make(map[string]int)
	for _, resultLine := range resultLinePattern.FindAllStringSubmatch(content, -1) {
		guesses, mentions := resultLine[1], resultLine[2]
		score := MaxScore

		if guesses != "X" {
			score, _ = strconv.Atoi(guesses)
		}

		for _, mention := range mentionPattern.FindAllStringSubmatch(mentions, -1) {
			userID := mention[1]
			scores[userID] = score
		}
	}

	return scores
}
