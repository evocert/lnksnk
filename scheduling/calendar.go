package scheduling

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evocert/lnksnk/iorw"
)

type Calendar struct {
	lckwkly   *sync.RWMutex
	weekly    map[time.Weekday][]*Interval
	lckdaily  *sync.RWMutex
	daily     []*Interval
	lckmnthly *sync.RWMutex
	monthly   map[time.Month]map[int][]*Interval
	tloc      *time.Location
}

var strtoday = map[string]time.Weekday{
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
	"sun": time.Sunday, "sunday": time.Sunday}

var strtomonth = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December}

func NewCalendar() (clndr *Calendar) {
	clndr = &Calendar{tloc: time.Local, lckwkly: &sync.RWMutex{}, weekly: map[time.Weekday][]*Interval{}, lckdaily: &sync.RWMutex{}, daily: []*Interval{}, lckmnthly: &sync.RWMutex{}, monthly: map[time.Month]map[int][]*Interval{}}
	return
}

func (clndr *Calendar) weeklyRLock() {
	if clndr != nil && clndr.lckwkly != nil {
		clndr.lckwkly.RLock()
	}
}

func (clndr *Calendar) weeklyRUnlock() {
	if clndr != nil && clndr.lckwkly != nil {
		clndr.lckwkly.RUnlock()
	}
}

func (clndr *Calendar) weeklyLock() {
	if clndr != nil && clndr.lckwkly != nil {
		clndr.lckwkly.Lock()
	}
}

func (clndr *Calendar) weeklyUnlock() {
	if clndr != nil && clndr.lckwkly != nil {
		clndr.lckwkly.Unlock()
	}
}

func (clndr *Calendar) dailyRLock() {
	if clndr != nil && clndr.lckwkly != nil {
		clndr.lckdaily.RLock()
	}
}

func (clndr *Calendar) dailyRUnlock() {
	if clndr != nil && clndr.lckdaily != nil {
		clndr.lckdaily.RUnlock()
	}
}

func (clndr *Calendar) dailyLock() {
	if clndr != nil && clndr.lckdaily != nil {
		clndr.lckdaily.Lock()
	}
}

func (clndr *Calendar) dailyUnlock() {
	if clndr != nil && clndr.lckdaily != nil {
		clndr.lckdaily.Unlock()
	}
}

func (clndr *Calendar) monthlyRLock() {
	if clndr != nil && clndr.lckmnthly != nil {
		clndr.lckmnthly.RLock()
	}
}

func (clndr *Calendar) monthlyRUnlock() {
	if clndr != nil && clndr.lckmnthly != nil {
		clndr.lckmnthly.RUnlock()
	}
}

func (clndr *Calendar) monthlyLock() {
	if clndr != nil && clndr.lckmnthly != nil {
		clndr.lckmnthly.Lock()
	}
}

func (clndr *Calendar) monthlyUnlock() {
	if clndr != nil && clndr.lckmnthly != nil {
		clndr.lckmnthly.Unlock()
	}
}

func toTime(v interface{}, layout string, tloc ...*time.Location) (tm time.Time, err error) {
	if s, _ := v.(string); s != "" {
		if tm, err = time.Parse(layout, s); err == nil {
			if len(tloc) > 0 && tloc[0] != nil {
				tm = ChangeLocation(tm, tloc[0])
			}
		}
	}
	return
}

func intervals(dlftintrval string, v interface{}, loc ...interface{}) (intrvls []*Interval, err error) {
	if vmp, _ := v.(map[string]interface{}); len(vmp) > 0 {
		v = []interface{}{vmp}
	}
	if varr, _ := v.([]interface{}); len(varr) > 0 {
		func() {
			intrvals := []*Interval{}
			tloc := time.Local
			if locl := len(loc); locl > 0 {
				if dtloc, _ := loc[0].(*time.Location); dtloc != nil {
					tloc = dtloc
				} else if stloc, _ := loc[0].(string); stloc != "" {
					if stloc = strings.TrimFunc(stloc, iorw.IsSpace); stloc != "" {
						if tloc, err = time.LoadLocation(stloc); err != nil {
							return
						}
					}
				}
			}
			for _, av := range varr {
				if dlmp, _ := av.(map[string]interface{}); len(dlmp) > 0 {
					frmd, frmdok := dlmp["from"]
					if frmdok {
						delete(dlmp, "from")
					}
					tod, todok := dlmp["to"]
					if todok {
						delete(dlmp, "to")
					}
					intrvd, intrvdok := dlmp["interval"]
					if intrvdok {
						delete(dlmp, "interval")
					}
					intrval := dlftintrval
					if intrvdok {
						if intrvds, _ := intrvd.(string); intrvds != "" {
							if intrvds = strings.TrimFunc(intrvds, iorw.IsSpace); intrvds != "" {
								intrval = intrvds
							}
						}
					}
					if frmdok && todok {
						if from, fromerr := toTime(frmd, time.TimeOnly, tloc); fromerr == nil {
							if to, toerr := toTime(tod, time.TimeOnly, tloc); toerr == nil {
								if intrvl, _ := time.ParseDuration(fmt.Sprint(intrval)); intrvl > 0 {
									intrvals = append(intrvals, &Interval{from: from, to: to, intrval: intrvl})
								}
							}
						}
					}
					if len(dlmp) > 0 {
						for k, v := range dlmp {
							if from, fromerr := toTime(k, time.TimeOnly, tloc); fromerr == nil {
								if to, toerr := toTime(v, time.TimeOnly, tloc); toerr == nil {
									if intrvl, _ := time.ParseDuration(intrval); intrvl > 0 {
										intrvals = append(intrvals, &Interval{from: from, to: to, intrval: intrvl})
									}
								}
							}
						}
					}
				}
			}
			if len(intrvals) > 0 {
				intrvls = rectifyToIntarvals(sortIntervals(intrvals...)...)
			}
		}()
	}
	return
}

func sortIntervals(intrvls ...*Interval) (sortedintrvls []*Interval) {
	if len(intrvls) > 1 {
		sort.Slice(intrvls, func(i, j int) bool {
			return intrvls[i].from.UnixNano() < intrvls[j].from.UnixNano()
		})
	}
	sortedintrvls = intrvls
	return
}

func (clndr *Calendar) ClearMonth(month string, days ...int) {
	if clndr != nil {
		if month = strings.ToLower(strings.TrimFunc(month, iorw.IsSpace)); month != "" {
			func() {
				clndr.monthlyLock()
				defer clndr.monthlyUnlock()
				if mnth, monthok := strtomonth[month]; monthok {
					var themonth = clndr.monthly[mnth]
					if themonth != nil {
						if len(days) == 0 {
							for d := range themonth {
								days = append(days, d)
							}
						}
						for _, d := range days {
							if _, dok := themonth[d]; dok {
								themonth[d] = nil
								delete(themonth, d)
							}
						}
					}
				}
			}()
		}
	}
}

func (clndr *Calendar) ClearWeek(days ...string) {
	if clndr != nil {
		dys := []time.Weekday{}
		for _, day := range days {
			if day = strings.ToLower(strings.TrimFunc(day, iorw.IsSpace)); day != "" {
				if d, dok := strtoday[day]; dok {
					dys = append(dys, d)
				}
			}
		}

		func() {
			clndr.weeklyLock()
			defer clndr.weeklyUnlock()
			if len(days) == 0 && len(dys) == 0 {
				for d := range clndr.weekly {
					dys = append(dys, d)
				}
			}
			if len(dys) > 0 {
				for _, d := range dys {
					if _, dok := clndr.weekly[d]; dok {
						clndr.weekly[d] = nil
						delete(clndr.weekly, d)
					}
				}
			}
		}()
	}
}

func (clndr *Calendar) ClearDaily(frmtimes ...string) {
	if clndr != nil {
		if len(frmtimes) == 0 {
			func() {
				clndr.dailyLock()
				defer clndr.dailyUnlock()
				clndr.daily = nil
			}()
		} else {
			tstfrms := []time.Time{}
			cntsfrm := map[string]bool{}
			for _, tm := range frmtimes {
				if tm != "" {
					if !cntsfrm[tm] {
						cntsfrm[tm] = true
						if t, terr := time.Parse(time.TimeOnly, tm); terr == nil {
							tstfrms = append(tstfrms, t)
						}
					} else {
						continue
					}
				}
			}
			if tstl, dlyl := len(tstfrms), func() int {
				clndr.dailyRLock()
				defer clndr.dailyRUnlock()
				return len(clndr.daily)
			}(); tstl > 0 {
				sort.Slice(tstfrms, func(i, j int) bool {
					return tstfrms[i].UnixNano() < tstfrms[j].UnixNano()
				})
				func() {
					clndr.dailyLock()
					defer clndr.dailyUnlock()
					tnow := time.Now()
					canCheck := true
					for canCheck {
						tsti := 0
						dlyi := 0
						canCheck = false
						for tsti < tstl {
							for dlyi < dlyl && tsti < tstl {
								if TodayNano(tnow, tstfrms[tsti]) == TodayNano(tnow, clndr.daily[dlyi].from) {
									clndr.daily = append(clndr.daily[:dlyi], clndr.daily[dlyi+1])
									dlyl--
									tstfrms = append(tstfrms[:tsti], tstfrms[:tsti+1]...)
									tstl--
									if !canCheck {
										canCheck = true
									}
								} else {
									dlyi++
								}
							}
							tsti++
						}
					}
				}()
			}
		}
	}
}

func checkTick(t1 time.Time, tnow time.Time, intrvls []*Interval) (valid bool, nexttime time.Time) {
	if intrvl := len(intrvls); intrvl > 0 && t1.UnixNano() < tnow.UnixNano() {
		lstin := intrvl - 1
		if minfrom, maxto, tnownano := Today(tnow, intrvls[0].from), Today(tnow, intrvls[lstin].to), tnow.UnixNano(); minfrom.UnixNano() <= tnownano && tnownano < maxto.UnixNano() {
			lstivr := intrvls[lstin]
			for un, ivr := range intrvls {
				from, to, lstfrom, lstto := Today(tnow, ivr.from), Today(tnow, ivr.to), Today(tnow, lstivr.from), Today(tnow, lstivr.to)
				{
					if frmu, tou, lstfrmu, lsttou, chkfrmto, chklstfrmto := from.UnixNano(), to.UnixNano(), lstfrom.UnixNano(), lstto.UnixNano(), from.UnixNano() <= tnownano && tnownano < to.UnixNano(), lstfrom.UnixNano() < tnownano && tnownano < lstto.UnixNano(); chkfrmto || chklstfrmto {
						frmu, tou = func() int64 {
							if chkfrmto {
								return frmu
							}
							return lstfrmu
						}(), func() int64 {
							if chkfrmto {
								return tou
							}
							return lsttou
						}()
						intrvltouse := func() time.Duration {
							if chkfrmto {
								return ivr.intrval
							}
							return lstivr.intrval
						}()
						valid = true
						tnowcnt := (tnownano - frmu) / int64(intrvltouse)
						if chkfrmto {
							nexttime = from.Add(time.Duration(tnowcnt+1) * intrvltouse)
							if un < intrvl-1 && nexttime.UnixNano() >= to.UnixNano() {
								nexttime = TodayNano(tnow, lstivr.from)
							}
						} else if chklstfrmto {
							nexttime = lstfrom.Add(time.Duration(tnowcnt+1) * intrvltouse)
							nexttime = Today(tnow, nexttime)
						}
						break
					} else if !chkfrmto && !chklstfrmto && un < intrvl-1 && to.UnixNano() <= tnownano && tnownano <= Today(tnow, intrvls[un+1].from).UnixNano() {
						nexttime = TodayNano(tnow, lstivr.from)
						break
					}
					if lstin--; un >= lstin {
						break
					}
				}
			}
		}
	}
	return
}

func (clndr *Calendar) Tick(prevt ...time.Time) (valid bool, nexttime time.Time) {
	if clndr != nil {
		tnow := ChangeLocation(time.Now(), clndr.tloc)
		t1 := func() time.Time {
			if len(prevt) > 0 {
				return ChangeLocation(TodayNano(prevt[0], prevt[0]), clndr.tloc)
			}
			return TodayNano(tnow, tnow)
		}()
		if valid, nexttime = checkTick(t1, tnow, sortIntervals(func() (intv []*Interval) {
			clndr.monthlyRLock()
			defer clndr.monthlyRUnlock()
			if mnth := clndr.monthly[t1.Month()]; len(mnth) > 0 {
				intv = mnth[t1.Day()]
			}
			return
		}()...)); valid {
			return
		}
		if valid, nexttime = checkTick(TodayNano(t1, t1), tnow, sortIntervals(func() (intv []*Interval) {
			clndr.weeklyRLock()
			defer clndr.weeklyRUnlock()
			intv = clndr.weekly[t1.Weekday()]
			return
		}()...)); valid {
			return
		}
		if valid, nexttime = checkTick(TodayNano(tnow, t1), tnow, sortIntervals(func() (intv []*Interval) {
			clndr.dailyRLock()
			defer clndr.dailyRUnlock()
			intv = append(intv, clndr.daily...)
			return
		}()...)); valid {
			return
		}
	}
	return
}

func (clndr *Calendar) SetLocation(tloc interface{}) (err error) {
	if clndr != nil {
		if dtloc, _ := tloc.(*time.Location); dtloc != nil {
			clndr.tloc = dtloc
		} else if stloc, _ := tloc.(string); stloc != "" {
			if stloc = strings.TrimFunc(stloc, iorw.IsSpace); stloc != "" {
				if dtloc, err = time.LoadLocation(stloc); err != nil {
					return
				}
				clndr.tloc = dtloc
			}
		}
	}

	return
}

func rectifyToIntarvals(intrvls ...*Interval) (intrvals []*Interval) {
	if intvlsL := len(intrvls); intvlsL > 1 {
		checked := true
		for checked {
			checked = false
			irn := 0
			for irn < intvlsL {
				if irn < intvlsL-1 && intrvls[irn].to.UnixNano() > intrvls[irn+1].from.UnixNano() {
					if !checked {
						checked = true
					}
					intrvls[irn].to = Today(intrvls[irn].to, intrvls[irn+1].from)
					irn++
				} else {
					irn++
				}
			}
		}
	}
	intrvals = append(intrvals, intrvls...)
	return
}

func (clndr *Calendar) Set(a ...interface{}) {
	if clndr != nil {
		for _, d := range a {
			if mpkv, _ := d.(map[string]interface{}); len(mpkv) > 0 {
				intrval := "10s"
				if dfltintrval, intrvok := mpkv["interval"]; intrvok {
					if intrvals := dfltintrval.(string); intrvals != "" {
						if intrvals = strings.TrimFunc(intrvals, iorw.IsSpace); intrvals != "" {
							intrval = intrvals
						}
					}
					delete(mpkv, "interval")
				}
				for k, v := range mpkv {
					k = strings.ToLower(k)
					if k == "daily" {
						if intrvals, intrvlserr := intervals(intrval, v, clndr.tloc); intrvlserr == nil && len(intrvals) > 0 {
							func() {
								clndr.dailyLock()
								defer clndr.dailyUnlock()
								clndr.daily = intrvals
							}()
						}
					} else if day, dayok := strtoday[k]; dayok {
						if intrvals, intrvlserr := intervals(intrval, v, clndr.tloc); intrvlserr == nil && len(intrvals) > 0 {
							func() {
								clndr.weeklyLock()
								defer clndr.weeklyUnlock()
								clndr.weekly[day] = intrvals
							}()
						}
					} else if month, monthok := strtomonth[k]; monthok {
						if mnthkv, _ := v.(map[string]interface{}); len(mnthkv) > 0 {
							for md, mdv := range mnthkv {
								if di, _ := strconv.ParseInt(md, 10, 64); di > 0 {
									if intrvals, intrvlserr := intervals(intrval, mdv, clndr.tloc); intrvlserr == nil && len(intrvals) > 0 {
										func() {
											clndr.monthlyLock()
											defer clndr.monthlyUnlock()
											var mnth = clndr.monthly[month]
											if mnth == nil {
												mnth = map[int][]*Interval{}
												clndr.monthly[month] = mnth
											}
											mnth[int(di)] = intrvals
										}()
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

/*func init() {
	GLOBALSCHEDULING().Set("TEST",
		map[string]interface{}{"interval": "10s",
			"daily": []interface{}{
				map[string]interface{}{
					"00:00:00": "07:12:00"},
				map[string]interface{}{
					"07:12:50": "07:13:50", "interval": "10s"},
				map[string]interface{}{
					"07:14:50": "23:59:59"},
			},
			"monday": []interface{}{
				map[string]interface{}{
					"from": "00:00:00", "to": "23:59:59"},
			},
		})

	schdlng := GLOBALSCHEDULING()

	schdlng.Calendar("TEST").ClearDaily("07:12:50")

	go func() {

		nxt1 := time.Now()
		tck := time.NewTicker(10 * time.Nanosecond)
		for {
			<-tck.C
			if vld, nxtt := schdlng.Tick(nxt1, "TEST"); vld {
				//tck.Reset(time.Duration(nxtt.UnixNano()) - time.Duration(nxt1.UnixNano()))
				nxt1 = TodayNano(nxtt, nxtt)
				fmt.Println(nxt1)
				fmt.Println(time.Now())
			}
		}

	}()

}*/
