// Package suggest guesses which apps someone is about to want, for the quick-open row under the empty bar.
//
// It is private by construction: it reads only the launch times Omni already keeps on this computer to rank search
// results (internal/state, at most 20 per app), keeps nothing of its own, and sends nothing anywhere. There is no
// tracking of windows, of what is in an app, or of anything opened outside Omni. Forgetting the usage history
// (Settings) empties what it reads.
package suggest

import (
	"math"
	"slices"
	"time"

	"omni/internal/rank"
)

// oneOff is how much an app opened only once counts against a habit: it can still fill a slot while there is little
// history, but any app opened twice or more comes first.
const oneOff = 0.25

// hourWidth is how far, in hours, a past launch's time of day still says something about now (a Gaussian's width).
const hourWidth = 1.5

// Rank orders the ids by how likely each is to be wanted at now and returns at most n of them. Each app is scored by its
// frecency (rank.Frecency: often and lately) lifted by how much of its use falls at this time of day, on this kind of day
// (a weekday or the weekend): an app opened every morning comes first in the morning, one opened every evening in the evening.
func Rank(uses map[string][]time.Time, now time.Time, n int) []string {
	type scored struct {
		id    string
		score float64
	}
	var all []scored
	for id, times := range uses {
		if len(times) == 0 {
			continue
		}
		score := rank.Frecency(times, now) * (0.35 + habit(times, now))
		if len(times) == 1 {
			score *= oneOff
		}
		all = append(all, scored{id, score})
	}
	slices.SortFunc(all, func(a, b scored) int {
		if a.score != b.score {
			if a.score > b.score {
				return -1
			}
			return 1
		}
		if a.id < b.id { // the same score: a stable order, so the row does not shuffle between openings
			return -1
		}
		return 1
	})
	out := make([]string, 0, min(n, len(all)))
	for _, s := range all[:min(n, len(all))] {
		out = append(out, s.id)
	}
	return out
}

// habit is the share (0 to 1) of an app's launches that happened near now's time of day on the same kind of day,
// recent launches counting more.
func habit(times []time.Time, now time.Time) float64 {
	var near, total float64
	for _, at := range times {
		age := max(now.Sub(at), 0)
		weight := math.Exp(-age.Hours() / (24 * 30)) // a launch from a month ago counts a third as much
		total += weight
		closeness := math.Exp(-math.Pow(hoursApart(at, now)/hourWidth, 2))
		if weekend(at) != weekend(now) {
			closeness *= 0.4
		}
		near += weight * closeness
	}
	if total == 0 {
		return 0
	}
	return near / total
}

// hoursApart is the distance between two clock times, going round midnight: 23:00 and 01:00 are two hours apart.
func hoursApart(a, b time.Time) float64 {
	clock := func(t time.Time) float64 { return float64(t.Hour()) + float64(t.Minute())/60 }
	d := math.Abs(clock(a) - clock(b))
	return math.Min(d, 24-d)
}

func weekend(t time.Time) bool { return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday }
