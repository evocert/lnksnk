package scheduling

import (
	"strings"
	"sync"
	"time"

	"github.com/lnksnk/lnksnk/iorw"
)

type Scheduling struct {
	clndrs *sync.Map
}

func NewScheduling(a ...interface{}) (schdlng *Scheduling) {
	schdlng = &Scheduling{}
	schdlng.Set(a...)
	return
}

func (schdlng *Scheduling) Calendar(clndrname string) (clndr *Calendar) {
	if clndrname = strings.TrimFunc(clndrname, iorw.IsSpace); clndrname != "" && schdlng != nil && schdlng.clndrs != nil {
		if clv, clvok := schdlng.clndrs.Load(clndrname); clvok {
			clndr, _ = clv.(*Calendar)
		}
	}
	return
}

func (schdlng *Scheduling) ClearMonth(clndrs []*Calendar, month string, days ...int) {
	for _, cldnr := range clndrs {
		cldnr.ClearMonth(month, days...)
	}
}

func (schdlng *Scheduling) ClearWeek(clndrs []*Calendar, days ...string) {
	for _, cldnr := range clndrs {
		cldnr.ClearWeek(days...)
	}
}

func (schdlng *Scheduling) ClearDaily(clndrs []*Calendar, frmtimes ...string) {
	for _, cldnr := range clndrs {
		cldnr.ClearDaily(frmtimes...)
	}
}

func (schdlng *Scheduling) Calendars(clndrnames ...string) (clndrs []*Calendar) {
	for _, clndrname := range clndrnames {
		if clndrname = strings.TrimFunc(clndrname, iorw.IsSpace); clndrname != "" && schdlng != nil && schdlng.clndrs != nil {
			if clv, clvok := schdlng.clndrs.Load(clndrname); clvok {
				if clndr, _ := clv.(*Calendar); clndr != nil {
					clndrs = append(clndrs, clndr)
				}
			}
		}
	}
	return
}

func (schdlng *Scheduling) Tick(a ...interface{}) (valid bool, nexttime time.Time) {
	prevt := []time.Time{}
	clndrnmes := map[string]bool{}
	for _, d := range a {
		if ds, _ := d.(string); ds != "" {
			if ds = strings.TrimFunc(ds, iorw.IsSpace); ds != "" {
				clndrnmes[ds] = true
			}
		} else if len(prevt) == 0 {
			if dt, dtok := d.(time.Time); dtok {
				prevt = append(prevt, dt)
			}
		}
	}
	if schdlng != nil && schdlng.clndrs != nil {
		clndrs := []*Calendar{}
		clndrnmsl := len(clndrnmes)
		schdlng.clndrs.Range(func(key, value any) (done bool) {
			k, _ := key.(string)
			if cldr, _ := value.(*Calendar); cldr != nil && (clndrnmsl == 0) || (clndrnmsl > 0 && clndrnmes[k]) {
				clndrs = append(clndrs, cldr)
			}
			return !done
		})
		prvu := int64(0)
		for _, clndr := range clndrs {
			if vld, nxtt := clndr.Tick(prevt...); vld {
				if !valid {
					valid = vld
				}
				if prvu < nxtt.UnixNano() {
					prvu = nxtt.UnixNano()
					nexttime = nxtt
				}
			}
		}
	}
	return
}

func (schdlng *Scheduling) Set(a ...interface{}) {
	if al := len(a); al > 0 && schdlng != nil {

		var clndr *Calendar = nil
		for _, d := range a {
			if ds, dsok := d.(string); dsok {
				clndr = nil
				if ds = strings.TrimFunc(ds, iorw.IsSpace); ds != "" {
					clndrs := func() *sync.Map {
						if schdlng.clndrs == nil {
							schdlng.clndrs = &sync.Map{}
						}
						return schdlng.clndrs
					}()
					if clv, clok := clndrs.Load(ds); clok {
						clndr = clv.(*Calendar)
					} else {
						clndr = NewCalendar()
						clndrs.Store(ds, clndr)
					}
				}
			} else if clndr != nil {
				clndr.Set(d)
			}
		}
	}
}

var glbscheduling *Scheduling = NewScheduling()

func GLOBALSCHEDULING() *Scheduling {
	return glbscheduling
}
