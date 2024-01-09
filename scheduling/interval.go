package scheduling

import (
	"time"
)

type Interval struct {
	from    time.Time
	to      time.Time
	intrval time.Duration
}

func (intrval *Interval) Valid(tloc *time.Location, prevtsttime ...time.Time) (valid bool, cantrigger bool, nxttsttime time.Time) {
	if intrval != nil {
		if from, to, tnow := ChangeLocation(Today(time.Now(), intrval.from), tloc), ChangeLocation(Today(time.Now(), intrval.to), tloc), ChangeLocation(time.Now(), tloc); from.UnixNano() < to.UnixNano() && from.UnixNano() <= tnow.UnixNano() && tnow.UnixNano() < to.UnixNano() {
			if tsttime := func() time.Time {
				if prvtl := len(prevtsttime); prvtl > 0 {
					prvt := prevtsttime[0]
					if prvt.Location() != tloc {
						prvt = ChangeLocation(prvt, tloc)
					}
					if prvt.UnixNano() <= from.UnixNano() {
						return Today(from, from)
					}
					return Today(prvt, prvt)
				} else {
					return Today(tnow, tnow)
				}
			}(); from.UnixNano() <= tsttime.UnixNano() && tsttime.UnixNano() < to.UnixNano() {
				if maxnowduration, maxduration, interval := tnow.UnixNano()-from.UnixNano(), tsttime.UnixNano()-from.UnixNano(), intrval.intrval.Nanoseconds(); maxduration > 0 && maxduration > interval {
					intrvcnt := maxduration / interval
					maxintrvcnt := maxnowduration / interval
					crntintrvtime := from.Add(time.Duration(intrvcnt * interval))
					crntmaxintrvtime := from.Add(time.Duration(maxintrvcnt * interval))
					if cantrigger = crntintrvtime.UnixNano() <= crntmaxintrvtime.UnixNano(); cantrigger {
						if crntintrvtime.Add(time.Duration(interval)).UnixNano() < to.UnixNano() {
							nxttsttime = crntintrvtime.Add(time.Duration(interval))
						} else {
							nxttsttime = Bod(tnow.Add(time.Hour * 24))
						}
					}
					valid = true
				}
			}
		}
	}
	return
}

func Bod(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func Today(tday time.Time, t time.Time) time.Time {
	year, month, day := tday.Date()
	return time.Date(year, month, day, t.Hour(), t.Minute(), t.Second(), 0, tday.Location())
}

func TodayNano(tday time.Time, t time.Time) time.Time {
	year, month, day := tday.Date()
	return time.Date(year, month, day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tday.Location())
}

func ChangeLocation(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}
