package wordlecount

// data storage structs for weekly and lifetime stats

// obj for the week itself not individual players
type Week struct {
	Start     string                 `json:"weekStart"`     // Monday, YYYY-MM-DD ? cant do M T W etc
	ChannelID string                 `json:"channelId"`     // good to future proof incase other servers use
	Days      []string               `json:"processedDays"` // keep track of which days have been processed?
	Players   map[string]*WeekPlayer `json:"players"`
}

// pplayers weekly stats
type WeekPlayer struct {
	Name       string         `json:"name"`
	Score      int            `json:"score"`
	DaysPlayed int            `json:"daysPlayed"`
	Results    map[string]int `json:"results"` // date -> guesses
}

// holds map for all time player stat and keeps track of weeks just gone
type Lifetime struct {
	LastWeek string                     `json:"lastWeekMerged"`
	Players  map[string]*LifetimePlayer `json:"players"`
}

// player's all time stats
type LifetimePlayer struct {
	Name          string  `json:"name"`
	DaysPlayed    int     `json:"daysPlayed"`
	WeeksPlayed   int     `json:"weeksPlayed"`
	CompleteWeeks int     `json:"completeWeeks"`
	TotalScore    int     `json:"totalScore"`
	AverageScore  float64 `json:"averageScore"`
}

// weekly leaderboard rank for a player
type Standing struct {
	Rank   int
	UserID string
	*WeekPlayer
}
