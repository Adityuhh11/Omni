package suggest

import (
	"slices"
	"testing"
	"time"
)

// at is a time on a weekday (Wednesday 2026-11-18) the given number of days earlier.
func at(daysAgo, hour int) time.Time {
	return time.Date(2026, 11, 18, hour, 0, 0, 0, time.Local).AddDate(0, 0, -daysAgo)
}

func TestTheAppUsedAtThisTimeOfDayComesFirst(t *testing.T) {
	uses := map[string][]time.Time{}
	for day := 1; day <= 5; day++ {
		uses["app:mail"] = append(uses["app:mail"], at(day*7, 9))    // Wednesday mornings
		uses["app:music"] = append(uses["app:music"], at(day*7, 20)) // Wednesday evenings
	}
	if got := Rank(uses, at(0, 9), 2); !slices.Equal(got, []string{"app:mail", "app:music"}) {
		t.Fatalf("in the morning: %v", got)
	}
	if got := Rank(uses, at(0, 20), 2); !slices.Equal(got, []string{"app:music", "app:mail"}) {
		t.Fatalf("in the evening: %v", got)
	}
}

func TestAnAppOpenedOnceComesAfterHabitsAndNIsTheMost(t *testing.T) {
	uses := map[string][]time.Time{
		"app:once": {at(0, 9)},
		"app:a":    {at(1, 9), at(2, 9)},
		"app:b":    {at(1, 10), at(2, 10)},
		"app:c":    {at(1, 11), at(2, 11)},
	}
	got := Rank(uses, at(0, 9), 2)
	if len(got) != 2 || slices.Contains(got, "app:once") {
		t.Fatalf("Rank = %v", got)
	}
	if got := Rank(map[string][]time.Time{"app:once": {at(0, 9)}}, at(0, 9), 3); !slices.Equal(got, []string{"app:once"}) {
		t.Fatalf("with little history a single launch still counts: %v", got)
	}
	if Rank(nil, at(0, 9), 4) == nil || len(Rank(nil, at(0, 9), 4)) != 0 {
		t.Fatal("no history should give an empty, non-nil list")
	}
}

func TestMidnightIsNearBothSides(t *testing.T) {
	late := time.Date(2026, 11, 18, 23, 30, 0, 0, time.Local)
	early := time.Date(2026, 11, 19, 0, 30, 0, 0, time.Local)
	if d := hoursApart(late, early); d != 1 {
		t.Fatalf("hoursApart = %v", d)
	}
}
