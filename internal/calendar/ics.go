package calendar

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"time"
)

// vevent is one VEVENT of a feed, as written: a recurring event is one vevent with a rule, and a changed occurrence of it is
// another vevent with the same uid and the start it replaces (recurrenceID).
type vevent struct {
	uid, summary, location, description, url, status string
	start, end                                       time.Time
	allDay                                           bool
	rrule                                            string
	exdates                                          []time.Time
	recurrenceID                                     time.Time
}

// parse reads the events of an iCalendar file (RFC 5545). What it does not need (alarms, attendees, time zone definitions) is
// skipped; a line it cannot read is skipped too, so one odd event does not cost the whole calendar.
func parse(data []byte, local *time.Location) []vevent {
	events := []vevent{} // never nil: an empty calendar is a calendar, and nil means a fetch that failed
	var cur *vevent
	depth := 0 // nesting inside the VEVENT (a VALARM in it has a DESCRIPTION of its own)
	for _, line := range unfold(data) {
		name, params, value, ok := split(line)
		if !ok {
			continue
		}
		switch {
		case name == "BEGIN" && value == "VEVENT":
			cur, depth = &vevent{}, 0
			continue
		case name == "END" && value == "VEVENT" && cur != nil:
			if !cur.start.IsZero() {
				if cur.end.IsZero() || !cur.end.After(cur.start) {
					cur.end = cur.start
					if cur.allDay {
						cur.end = cur.start.AddDate(0, 0, 1)
					}
				}
				events = append(events, *cur)
			}
			cur = nil
			continue
		case cur == nil:
			continue
		case name == "BEGIN":
			depth++
			continue
		case name == "END":
			depth--
			continue
		case depth > 0:
			continue
		}
		switch name {
		case "UID":
			cur.uid = value
		case "SUMMARY":
			cur.summary = unescape(value)
		case "LOCATION":
			cur.location = unescape(value)
		case "DESCRIPTION":
			cur.description = unescape(value)
		case "URL":
			cur.url = value
		case "STATUS":
			cur.status = strings.ToUpper(value)
		case "DTSTART":
			cur.start, cur.allDay, _ = parseTime(value, params, local)
		case "DTEND":
			cur.end, _, _ = parseTime(value, params, local)
		case "DURATION":
			if d, ok := parseDuration(value); ok && !cur.start.IsZero() {
				cur.end = cur.start.Add(d)
			}
		case "RRULE":
			cur.rrule = value
		case "EXDATE":
			for v := range strings.SplitSeq(value, ",") {
				if t, _, ok := parseTime(v, params, local); ok {
					cur.exdates = append(cur.exdates, t)
				}
			}
		case "RECURRENCE-ID":
			cur.recurrenceID, _, _ = parseTime(value, params, local)
		}
	}
	return events
}

// unfold joins the lines a feed folded: a line that starts with a space or a tab goes on the one before.
func unfold(data []byte) []string {
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// split cuts "DTSTART;TZID=Europe/Berlin:20261103T100000" into its name, parameters and value. A colon inside a quoted
// parameter value does not end the parameters.
func split(line string) (name string, params map[string]string, value string, ok bool) {
	quoted, colon := false, -1
	for i, r := range line {
		if r == '"' {
			quoted = !quoted
		} else if r == ':' && !quoted {
			colon = i
			break
		}
	}
	if colon < 0 {
		return "", nil, "", false
	}
	head := strings.Split(line[:colon], ";")
	params = map[string]string{}
	for _, p := range head[1:] {
		if k, v, ok := strings.Cut(p, "="); ok {
			params[strings.ToUpper(k)] = strings.Trim(v, `"`)
		}
	}
	return strings.ToUpper(head[0]), params, line[colon+1:], true
}

func unescape(s string) string {
	return strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`).Replace(s)
}

// parseTime reads a DATE ("20261103", a day: all day, at midnight here), a UTC time ("20261103T100000Z"), or a time in the zone its
// TZID names. ponytail: a zone Go does not know by name (Outlook writes Windows names such as "Pacific Standard Time") is read as
// local time; reading the feed's VTIMEZONE would fix that.
func parseTime(value string, params map[string]string, local *time.Location) (t time.Time, allDay bool, ok bool) {
	value = strings.TrimSpace(value)
	if params["VALUE"] == "DATE" || len(value) == 8 {
		t, err := time.ParseInLocation("20060102", value, local)
		return t, true, err == nil
	}
	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		return t, false, err == nil
	}
	loc := local
	if tz := params["TZID"]; tz != "" {
		if l, err := time.LoadLocation(strings.TrimPrefix(tz, "/")); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation("20060102T150405", value, loc)
	return t, false, err == nil
}

// parseDuration reads an iCalendar duration such as "PT1H30M", "P1D" or "P1W".
func parseDuration(s string) (time.Duration, bool) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "+"), "P")
	var d time.Duration
	inTime := false
	num := ""
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			num += string(r)
		case r == 'T':
			inTime = true
		default:
			n, err := strconv.Atoi(num)
			if err != nil {
				return 0, false
			}
			num = ""
			switch {
			case r == 'W':
				d += time.Duration(n) * 7 * 24 * time.Hour
			case r == 'D':
				d += time.Duration(n) * 24 * time.Hour
			case r == 'H' && inTime:
				d += time.Duration(n) * time.Hour
			case r == 'M' && inTime:
				d += time.Duration(n) * time.Minute
			case r == 'S' && inTime:
				d += time.Duration(n) * time.Second
			default:
				return 0, false
			}
		}
	}
	return d, num == ""
}
