package wordlecount

// What gets saved for each server. Week is weekly_scores.json and Lifetime is
// lifetime_tally.json, players are keyed by their Discord user ID.

// Week is the running tally for the current Monday–Sunday week.
type Week struct {
	Start         string                 `json:"weekStart"`     // Monday, YYYY-MM-DD
	ChannelID     string                 `json:"channelId"`     // where the leaderboard gets posted
	ProcessedDays []string               `json:"processedDays"` // so the same day isn't counted twice
	Players       map[string]*WeekPlayer `json:"players"`
}

// WeekPlayer is one player's stats for the week so far.
type WeekPlayer struct {
	Name       string         `json:"name"`
	Score      int            `json:"score"`
	DaysPlayed int            `json:"daysPlayed"`
	Results    map[string]int `json:"results"` // date -> guesses
}

// Lifetime holds everyone's all-time stats. Weeks are added once they finish,
// LastWeekMerged is the Monday of the last one added.
type Lifetime struct {
	LastWeekMerged string                     `json:"lastWeekMerged"`
	Players        map[string]*LifetimePlayer `json:"players"`
}

// LifetimePlayer is one player's all-time stats.
type LifetimePlayer struct {
	Name          string  `json:"name"`
	DaysPlayed    int     `json:"daysPlayed"`
	WeeksPlayed   int     `json:"weeksPlayed"`
	WeeksWon      int     `json:"weeksWon"`
	CompleteWeeks int     `json:"completeWeeks"`
	TotalScore    int     `json:"totalScore"`
	AverageScore  float64 `json:"averageScore"`
}

// Standing is a player's place on the weekly leaderboard. Not saved, it's
// worked out each time.
type Standing struct {
	Rank   int
	UserID string
	*WeekPlayer
}
