package enviro

import (
	"fmt"
	"time"
)

type timeFormatType int

const (
	timeFormatNoTimezone timeFormatType = iota
	timeFormatNumericTimezone
	timeFormatNamedTimezone
	timeFormatNumericAndNamedTimezone
	timeFormatTimeOnly
)

type timeFormat struct {
	format string
	typ    timeFormatType
}

var timeFormats = []timeFormat{
	// Keep common formats at the top.
	{"2006-01-02", timeFormatNoTimezone},
	{time.RFC3339, timeFormatNumericTimezone},
	{"2006-01-02T15:04:05", timeFormatNoTimezone}, // iso8601 without timezone
	{time.RFC1123Z, timeFormatNumericTimezone},
	{time.RFC1123, timeFormatNamedTimezone},
	{time.RFC822Z, timeFormatNumericTimezone},
	{time.RFC822, timeFormatNamedTimezone},
	{time.RFC850, timeFormatNamedTimezone},
	{"2006-01-02 15:04:05.999999999 -0700 MST", timeFormatNumericAndNamedTimezone}, // Time.String()
	{"2006-01-02T15:04:05-0700", timeFormatNumericTimezone},                        // RFC3339 without timezone hh:mm colon
	{"2006-01-02 15:04:05Z0700", timeFormatNumericTimezone},                        // RFC3339 without T or timezone hh:mm colon
	{"2006-01-02 15:04:05", timeFormatNoTimezone},
	{time.ANSIC, timeFormatNoTimezone},
	{time.UnixDate, timeFormatNamedTimezone},
	{time.RubyDate, timeFormatNumericTimezone},
	{"2006-01-02 15:04:05Z07:00", timeFormatNumericTimezone},
	{"02 Jan 2006", timeFormatNoTimezone},
	{"2006-01-02 15:04:05 -07:00", timeFormatNumericTimezone},
	{"2006-01-02 15:04:05 -0700", timeFormatNumericTimezone},
	{time.Kitchen, timeFormatTimeOnly},
	{time.Stamp, timeFormatTimeOnly},
	{time.StampMilli, timeFormatTimeOnly},
	{time.StampMicro, timeFormatTimeOnly},
	{time.StampNano, timeFormatTimeOnly},
}

func parseDateWith(s string, location *time.Location, formats []timeFormat) (d time.Time, e error) {
	for _, format := range formats {
		if d, e = time.Parse(format.format, s); e == nil {
			// Some time formats have a zone name, but no offset, so it gets
			// put in that zone name (not the default one passed in to us), but
			// without that zone's offset. So set the location manually.
			if format.typ <= timeFormatNamedTimezone {
				if location == nil {
					location = time.Local
				}
				year, month, day := d.Date()
				hour, min, sec := d.Clock()
				d = time.Date(year, month, day, hour, min, sec, d.Nanosecond(), location)
			}

			return
		}
	}
	return d, fmt.Errorf("unable to parse date: %s", s)
}
