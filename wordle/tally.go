package wordlecount

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// -- DATE TIME HANDLERS

const dateLayout = "2006-01-02"
const dayCutoffHour = 6

// DayIndex returns how many days date is after the week's start, with Monday = 0.
func (week *Week) DayIndex(date string) int {
	start, _ := time.Parse(dateLayout, week.Start)
	parsedDate, _ := time.Parse(dateLayout, date)
	return int(parsedDate.Sub(start).Hours() / 24)
}

// PostDay returns the date (YYYY-MM-DD in loc) a results post posted at t
// belongs to.
func PostDay(t time.Time, loc *time.Location) string {
	t = t.In(loc)
	if t.Hour() < dayCutoffHour {
		t = t.AddDate(0, 0, -1)
	}
	return t.Format(dateLayout)
}

// WeekStart returns the Monday on or before date.
func WeekStart(date string) string {
	d, _ := time.Parse(dateLayout, date)
	return d.AddDate(0, 0, -(int(d.Weekday())+6)%7).Format(dateLayout)
}

// -- DATE TIME HANDLERS END

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
	week.backdate(elapsed)
	return true
}

func (week *Week) backdate(elapsed int) {
	for _, player := range week.Players {
		player.DaysPlayed = len(player.Results)
		player.Score = sum(player.Results) + MaxScore*(elapsed-player.DaysPlayed)
	}
}

// Standings ranks the week's players, lowest score first. Ties share a rank
// and are ordered by days played, then name.
func (w *Week) Standings() []Standing {
	out := make([]Standing, 0, len(w.Players))

	for id, p := range w.Players {
		out = append(out, Standing{UserID: id, WeekPlayer: p})
	}

	slices.SortFunc(out, func(a, b Standing) int {
		return cmp.Or(
			cmp.Compare(a.Score, b.Score),
			cmp.Compare(b.DaysPlayed, a.DaysPlayed),
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.UserID, b.UserID),
		)
	})

	for i := range out {
		out[i].Rank = i + 1
		if i > 0 && out[i].Score == out[i-1].Score {
			out[i].Rank = out[i-1].Rank
		}
	}

	return out
}

// Finish scores the week over all seven days and adds it to lifetime. It
// returns false without changing lifetime if this week was already merged.
func (w *Week) Finish(lifetime *Lifetime) bool {
	w.backdate(DaysPerWeek)

	if lifetime.LastWeek >= w.Start {
		return false
	}

	lifetime.LastWeek = w.Start
	if lifetime.Players == nil {
		lifetime.Players = make(map[string]*LifetimePlayer)
	}

	for id, p := range w.Players {
		lp, ok := lifetime.Players[id]

		if !ok {
			lp = &LifetimePlayer{}
			lifetime.Players[id] = lp
		}

		if p.Name != "" {
			lp.Name = p.Name
		}

		lp.DaysPlayed += p.DaysPlayed
		lp.WeeksPlayed++
		if p.DaysPlayed == DaysPerWeek {
			lp.CompleteWeeks++
		}

		lp.TotalScore += sum(p.Results)
		if lp.DaysPlayed > 0 {
			lp.AverageScore = math.Round(float64(lp.TotalScore)/float64(lp.DaysPlayed)*100) / 100
		}
	}

	return true
}

// sum returns the total of all values in the map.
func sum(m map[string]int) int {
	total := 0
	for _, v := range m {
		total += v
	}
	return total
}
