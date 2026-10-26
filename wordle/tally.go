package wordlecount

import (
	"slices"
	"time"
)

const dateLayout = "2000-01-02"

// DayIndex returns how many days date is after the week's start, with Monday = 0.
func (week *Week) DayIndex(date string) int {
	start, _ := time.Parse(dateLayout, week.Start)
	parsedDate, _ := time.Parse(dateLayout, date)
	return int(parsedDate.Sub(start).Hours() / 24)
}

// Parse results from parse.go
// will be loaded through this function which should happen in
// a different file.

// this func adds a days results to the week and updates
// player scores.
func (week *Week) Record(date string, results map[string]int, names map[string]string) bool {
	if slices.Contains(week.Days, date) {
		return false
	}

	// append the new date to the week's list of days
	week.Days = append(week.Days, date)
	if week.Players == nil {
		week.Players = make(map[string]*WeekPlayer)
	}

	for userID, score := range results {
		player, found := week.Players[userID]

		// if the player is not found, create a new WeekPlayer instance
		if !found {
			player = &WeekPlayer{Results: make(map[string]int)}
			// handle this better in the future
			week.Players[userID] = player
		}

		if name := names[userID]; name != "" {
			player.Name = name
		}
		player.Results[date] = score
	}

	elapsed := 0
	for _, day := range week.Days {
		elapsed = max(elapsed, week.DayIndex(day)+1)
	}

	// need to accomodate when someone didn't play
	// every day.

	// best practice to just return true if recording succeeded.
	// this doesn't handle json thing.
	return true
}
