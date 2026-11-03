package wordlecount

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// -- DATE TIME HANDLERS

// YYYY-MM-DD, written with Go's reference date.
const dateLayout = "2006-01-02"

// Posts before 6am count towards the day before, as the Wordle app sometimes
// posts a bit after midnight.
const dayCutoffHour = 6

// DayIndex returns how many days date is after the week's start, with Monday = 0.
func (week *Week) DayIndex(date string) int {
	weekStart, _ := time.Parse(dateLayout, week.Start)
	parsedDate, _ := time.Parse(dateLayout, date)
	return int(parsedDate.Sub(weekStart).Hours() / 24)
}

// PostDay returns the date (YYYY-MM-DD in timezone) that a results post sent
// at postedAt belongs to.
func PostDay(postedAt time.Time, timezone *time.Location) string {
	localTime := postedAt.In(timezone)
	if localTime.Hour() < dayCutoffHour {
		localTime = localTime.AddDate(0, 0, -1)
	}
	return localTime.Format(dateLayout)
}

// WeekStart returns the Monday on or before date.
func WeekStart(date string) string {
	parsedDate, _ := time.Parse(dateLayout, date)
	daysSinceMonday := (int(parsedDate.Weekday()) + 6) % 7
	return parsedDate.AddDate(0, 0, -daysSinceMonday).Format(dateLayout)
}

// -- DATE TIME HANDLERS END

// Record adds a day's results to the week and re-scores everyone. It returns
// false and changes nothing if that day is already in.
func (week *Week) Record(date string, scores map[string]int, displayNames map[string]string) bool {
	if slices.Contains(week.ProcessedDays, date) {
		return false
	}

	// append the new date to the week's list of days
	week.ProcessedDays = append(week.ProcessedDays, date)
	if week.Players == nil {
		week.Players = make(map[string]*WeekPlayer)
	}

	for userID, score := range scores {
		player, found := week.Players[userID]

		// if the player is not found, create a new WeekPlayer instance
		if !found {
			player = &WeekPlayer{Results: make(map[string]int)}
			// handle this better in the future
			week.Players[userID] = player
		}

		if displayName := displayNames[userID]; displayName != "" {
			player.Name = displayName
		}
		player.Results[date] = score
	}

	// days so far is up to the latest day recorded, missed ones get charged
	daysElapsed := 0
	for _, day := range week.ProcessedDays {
		daysElapsed = max(daysElapsed, week.DayIndex(day)+1)
	}

	week.backdate(daysElapsed)
	return true
}

// backdate scores every player over the first daysElapsed days of the week,
// with each day they didn't play counting as MaxScore. Someone joining mid
// week starts off behind because of this.
func (week *Week) backdate(daysElapsed int) {
	for _, player := range week.Players {
		player.DaysPlayed = len(player.Results)
		player.Score = sum(player.Results) + MaxScore*(daysElapsed-player.DaysPlayed)
	}
}

// Standings ranks the week's players, lowest score first. Ties share a rank
// and the next rank follows straight on, so three people tied 2nd are
// followed by 3rd rather than 5th.
func (week *Week) Standings() []Standing {
	standings := make([]Standing, 0, len(week.Players))

	for userID, player := range week.Players {
		standings = append(standings, Standing{UserID: userID, WeekPlayer: player})
	}

	// ties go to whoever played more days, name and ID just keep the order stable
	slices.SortFunc(standings, func(first, second Standing) int {
		return cmp.Or(
			cmp.Compare(first.Score, second.Score),
			cmp.Compare(second.DaysPlayed, first.DaysPlayed),
			cmp.Compare(first.Name, second.Name),
			cmp.Compare(first.UserID, second.UserID),
		)
	})

	for position := range standings {
		switch {
		case position == 0:
			standings[position].Rank = 1
		case standings[position].Score == standings[position-1].Score:
			standings[position].Rank = standings[position-1].Rank
		default:
			standings[position].Rank = standings[position-1].Rank + 1
		}
	}

	return standings
}

// Finish scores the week over all seven days and adds it to lifetime. It
// returns false without changing lifetime if this week was already merged.
func (week *Week) Finish(lifetime *Lifetime) bool {
	week.backdate(DaysPerWeek)

	// Already merged, adding it again would double count everything.
	if lifetime.LastWeekMerged >= week.Start {
		return false
	}

	lifetime.LastWeekMerged = week.Start
	if lifetime.Players == nil {
		lifetime.Players = make(map[string]*LifetimePlayer)
	}

	// Everyone tied for 1st counts as having won the week.
	winners := make(map[string]bool)
	for _, standing := range week.Standings() {
		if standing.Rank == 1 {
			winners[standing.UserID] = true
		}
	}

	for userID, player := range week.Players {
		lifetimePlayer, found := lifetime.Players[userID]

		if !found {
			lifetimePlayer = &LifetimePlayer{}
			lifetime.Players[userID] = lifetimePlayer
		}

		if player.Name != "" {
			lifetimePlayer.Name = player.Name
		}

		lifetimePlayer.DaysPlayed += player.DaysPlayed
		lifetimePlayer.WeeksPlayed++
		if winners[userID] {
			lifetimePlayer.WeeksWon++
		}
		if player.DaysPlayed == DaysPerWeek {
			lifetimePlayer.CompleteWeeks++
		}

		lifetimePlayer.TotalScore += sum(player.Results)
		if lifetimePlayer.DaysPlayed > 0 {
			average := float64(lifetimePlayer.TotalScore) / float64(lifetimePlayer.DaysPlayed)
			lifetimePlayer.AverageScore = math.Round(average*100) / 100
		}
	}

	return true
}

// sum returns the total of all the scores in the map.
func sum(scores map[string]int) int {
	total := 0
	for _, score := range scores {
		total += score
	}
	return total
}
