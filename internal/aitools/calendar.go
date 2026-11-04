package aitools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"omni/internal/calendar"
	"omni/internal/chat"
)

// Calendar is the person's calendars; *calendar.Calendar satisfies it.
type Calendar interface {
	Between(from, to time.Time) []calendar.Event
}

const (
	maxCalendarDays = 14
	calendarLines   = 80 // events written for the model; a busy fortnight fits, a feed of thousands does not flood the context
	// Free time is counted inside working hours. ponytail: fixed 9:00 to 18:00; a setting when someone's day differs.
	dayOpens  = 9
	dayCloses = 18
	minFree   = 15 * time.Minute // a gap shorter than this is not time anyone can use
)

func (d Deps) calendarTool() chat.Local {
	return chat.Local{
		Group: "calendar", GroupTitle: "Calendar", Busy: "Reading your calendar",
		Def: chat.ObjectTool("calendar_events", "The person's calendar events and free time for some days, day by day. "+
			"Free time is worked out for you, between 9:00 and 18:00.", map[string]string{
			"when": "today, tomorrow, this week, next week, a weekday such as friday, or a date as 2026-11-20",
			"days": "how many days from that day, 1 to 14; leave it out for this week and next week",
		}, "when"),
		Prompt: "You can read the person's calendars with calendar_events: use it for their schedule, meetings, deadlines or free time. " +
			"Answer from what it returns; its free time is already worked out, so do not count it again.",
		Run: d.calendarEvents,
	}
}

func (d Deps) calendarEvents(_ context.Context, args map[string]string) (string, []chat.Source, error) {
	now := d.Now().Local()
	from, days, ok := calendarSpan(now, args["when"])
	if !ok {
		return "Error: “" + args["when"] + "” is not a day I know. Use today, tomorrow, this week, next week, a weekday or a date as 2026-11-20.", nil, nil
	}
	// A small model fills in days even for a week ("this week", days 1); only a single day's length can be changed.
	if n, err := strconv.Atoi(strings.Trim(args["days"], `" `)); err == nil && n > 0 && days == 1 {
		days = min(n, maxCalendarDays)
	}
	to := from.AddDate(0, 0, days)
	events := d.Calendar.Between(from, to)

	var b strings.Builder
	fmt.Fprintf(&b, "Calendar from %s to %s. It is now %s. Times are local, 24-hour.\n", from.Format("Monday 2 January"),
		to.AddDate(0, 0, -1).Format("Monday 2 January 2006"), now.Format("Monday 15:04"))
	var sources []chat.Source
	written := 0
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		next := day.AddDate(0, 0, 1)
		b.WriteString("\n" + day.Format("Monday 2 January") + ":\n")
		var timed []calendar.Event
		for _, e := range events {
			if !e.Start.Before(next) || !e.End.After(day) {
				continue
			}
			if !e.AllDay {
				timed = append(timed, e)
			}
			if written < calendarLines {
				b.WriteString("- " + eventLine(e, day) + "\n")
				written++
			}
			if !e.Start.Before(day) || day.Equal(from) { // once, on its first day in the span
				sources = append(sources, eventSource(e))
			}
		}
		if len(timed) == 0 && !hasAllDay(events, day, next) {
			b.WriteString("- no events\n")
		}
		if free := freeTime(day, now, timed); free != "" {
			b.WriteString("Free: " + free + "\n")
		}
	}
	if written == calendarLines {
		b.WriteString("\n(More events than shown; ask about fewer days.)\n")
	}
	return b.String(), sources, nil
}

// calendarSpan reads "when" as a first day and a number of days.
func calendarSpan(now time.Time, when string) (time.Time, int, bool) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	when = strings.ToLower(strings.TrimSpace(when))
	switch when {
	case "", "today", "now":
		return today, 1, true
	case "tomorrow":
		return today.AddDate(0, 0, 1), 1, true
	case "this week", "week":
		return today, int(monday.AddDate(0, 0, 7).Sub(today).Round(24*time.Hour) / (24 * time.Hour)), true
	case "next week":
		return monday.AddDate(0, 0, 7), 7, true
	}
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		if name := strings.ToLower(wd.String()); when == name || when == "next "+name || when == "on "+name {
			return today.AddDate(0, 0, (int(wd)-int(today.Weekday())+7)%7), 1, true
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", when, now.Location()); err == nil {
		return t, 1, true
	}
	return time.Time{}, 0, false
}

func eventLine(e calendar.Event, day time.Time) string {
	title := e.Title
	if title == "" {
		title = "(no title)"
	}
	var when string
	switch {
	case e.AllDay:
		when = "all day"
	case e.Start.Before(day):
		when = "until " + e.End.Format("15:04")
	default:
		when = e.Start.Format("15:04") + "–" + e.End.Format("15:04")
	}
	line := when + " " + title + " (" + e.Calendar + ")"
	if e.Location != "" && !strings.HasPrefix(e.Location, "http") {
		line += ", at " + e.Location
	}
	if e.Join != "" {
		line += ", online meeting"
	}
	return line
}

func hasAllDay(events []calendar.Event, day, next time.Time) bool {
	for _, e := range events {
		if e.AllDay && e.Start.Before(next) && e.End.After(day) {
			return true
		}
	}
	return false
}

// freeTime is the gaps of at least minFree between the day's timed events inside working hours, from now on for today, as
// "9:00–10:30, 14:00–18:00"; "none" if there are none, "" for a day that is over.
func freeTime(day, now time.Time, events []calendar.Event) string {
	opens := time.Date(day.Year(), day.Month(), day.Day(), dayOpens, 0, 0, 0, day.Location())
	closes := time.Date(day.Year(), day.Month(), day.Day(), dayCloses, 0, 0, 0, day.Location())
	if now.After(opens) {
		opens = now.Truncate(time.Minute)
	}
	if !opens.Before(closes) {
		return ""
	}
	var gaps []string
	at := opens
	for _, e := range events { // Between gives them first to last
		if !e.Start.Before(closes) {
			break
		}
		if e.Start.Sub(at) >= minFree {
			gaps = append(gaps, at.Format("15:04")+"–"+e.Start.Format("15:04"))
		}
		if e.End.After(at) {
			at = e.End
		}
	}
	if closes.Sub(at) >= minFree {
		gaps = append(gaps, at.Format("15:04")+"–"+closes.Format("15:04"))
	}
	if len(gaps) == 0 {
		return "none"
	}
	return strings.Join(gaps, ", ")
}

func eventSource(e calendar.Event) chat.Source {
	return chat.Source{Title: e.Title, Event: &chat.Event{ID: e.ID, Start: e.Start.UnixMilli(), End: e.End.UnixMilli(), AllDay: e.AllDay,
		Calendar: e.Calendar, Color: e.Color, Join: e.Join}}
}
