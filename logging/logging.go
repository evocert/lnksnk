package logging

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/resources"
)

type Logger interface {
	Error(v ...interface{}) error
	Warning(v ...interface{}) error
	Info(v ...interface{}) error

	Errorf(format string, a ...interface{}) error
	Warningf(format string, a ...interface{}) error
	Infof(format string, a ...interface{}) error

	Handler(a ...interface{}) Logger
	Active() bool
}

type loggingwrap struct {
	lggng *logging
}

func (lggngwrp *loggingwrap) Active() (active bool) {
	if lggngwrp != nil {
		active = lggngwrp.lggng != nil
	}
	return
}

func (lggngwrp *loggingwrap) Error(v ...interface{}) (err error) {
	if lggngwrp != nil && lggngwrp.Active() && lggngwrp.lggng != nil {
		err = lggngwrp.lggng.log(LogError, v...)
	}
	return
}

func (lggngwrp *loggingwrap) Warning(v ...interface{}) (err error) {
	if lggngwrp != nil && lggngwrp.Active() && lggngwrp.lggng != nil {
		err = lggngwrp.lggng.log(LogWarning, v...)
	}
	return
}

type logginghandler struct {
	infocall   func(...interface{}) error
	infofcall  func(string, ...interface{}) error
	warncall   func(...interface{}) error
	warnfcall  func(string, ...interface{}) error
	errorcall  func(...interface{}) error
	errorfcall func(string, ...interface{}) error
	activecall func() bool
}

func (lggnghndl *logginghandler) Handler(a ...interface{}) (lggng Logger) {
	var infocall func(...interface{}) error = nil
	var infofcall func(string, ...interface{}) error = nil
	var warncall func(...interface{}) error = nil
	var warnfcall func(string, ...interface{}) error = nil
	var errorcall func(...interface{}) error = nil
	var errorfcall func(string, ...interface{}) error = nil

	for len(a) > 0 && len(a)%2 == 0 {
		if lgtpe, lgtpeok := a[0].(logtype); lgtpeok {
			dlgfcall, _ := a[1].(func(string, ...interface{}) error)
			dlgcall, _ := a[1].(func(...interface{}) error)
			switch lgtpe {
			case LogInfo:
				if dlgcall != nil && infocall == nil {
					infocall = dlgcall
				} else if dlgfcall != nil && infofcall == nil {
					infofcall = dlgfcall
				}
			case LogWarning:
				if dlgcall != nil && warncall == nil {
					warncall = dlgcall
				} else if dlgfcall != nil && warnfcall == nil {
					warnfcall = dlgfcall
				}
			case LogError:
				if dlgcall != nil && errorcall == nil {
					errorcall = dlgcall
				} else if dlgfcall != nil && errorfcall == nil {
					errorfcall = dlgfcall
				}
			default:
				a = nil
			}
			if len(a) >= 2 && len(a)%2 == 0 {
				a = a[2:]
			} else {
				break
			}
		} else {
			break
		}
	}

	if infocall == nil {
		infocall = func(a ...interface{}) error {
			return lggnghndl.Info(a...)
		}
	}
	if infofcall == nil {
		infofcall = func(format string, a ...interface{}) error {
			return lggnghndl.Infof(format, a...)
		}
	}

	if warncall == nil {
		warncall = func(a ...interface{}) error {
			return lggnghndl.Warning(a...)
		}
	}
	if warnfcall == nil {
		warnfcall = func(format string, a ...interface{}) error {
			return lggnghndl.Warningf(format, a...)
		}
	}

	if errorcall == nil {
		errorcall = func(a ...interface{}) error {
			return lggnghndl.Error(a...)
		}
	}
	if errorfcall == nil {
		errorfcall = func(format string, a ...interface{}) error {
			return lggnghndl.Errorf(format, a...)
		}
	}
	lggng = &logginghandler{activecall: func() (active bool) {
		return lggnghndl.Active()
	}, infocall: infocall, infofcall: infofcall, warncall: warncall, warnfcall: warnfcall, errorcall: errorcall, errorfcall: errorfcall}

	return
}

func (lggnghndl *logginghandler) Active() (active bool) {
	if lggnghndl != nil && lggnghndl.activecall != nil {
		active = lggnghndl.activecall()
	}
	return
}

func (lggnghndl *logginghandler) Info(v ...interface{}) (err error) {
	if lggnghndl != nil && lggnghndl.Active() && lggnghndl.infocall != nil {
		err = lggnghndl.infocall(v...)
	}
	return
}

func (lggnghndl *logginghandler) Warning(v ...interface{}) (err error) {
	if lggnghndl != nil && lggnghndl.Active() && lggnghndl.warncall != nil {
		err = lggnghndl.warncall(v...)
	}
	return
}

func (lggnghndl *logginghandler) Error(v ...interface{}) (err error) {
	if lggnghndl != nil && lggnghndl.Active() && lggnghndl.errorcall != nil {
		err = lggnghndl.errorcall(v...)
	}
	return
}

func (lggnghndl *logginghandler) Infof(format string, v ...interface{}) (err error) {
	if lggnghndl != nil && lggnghndl.Active() && lggnghndl.infofcall != nil {
		err = lggnghndl.infofcall(format, v...)
	}
	return
}

func (lggnghndl *logginghandler) Warningf(format string, v ...interface{}) (err error) {
	if lggnghndl != nil && lggnghndl.Active() && lggnghndl.warnfcall != nil {
		err = lggnghndl.warnfcall(format, v...)
	}
	return
}

func (lggnghndl *logginghandler) Errorf(format string, v ...interface{}) (err error) {
	if lggnghndl != nil && lggnghndl.Active() && lggnghndl.errorfcall != nil {
		err = lggnghndl.errorfcall(format, v...)
	}
	return
}

func (lggngwrp *loggingwrap) Handler(a ...interface{}) (lggng Logger) {
	var infocall func(...interface{}) error = nil
	var infofcall func(string, ...interface{}) error = nil
	var warncall func(...interface{}) error = nil
	var warnfcall func(string, ...interface{}) error = nil
	var errorcall func(...interface{}) error = nil
	var errorfcall func(string, ...interface{}) error = nil

	for len(a) > 0 && len(a)%2 == 0 {
		if lgtpe, lgtpeok := a[0].(logtype); lgtpeok {
			dlgfcall, _ := a[1].(func(string, ...interface{}) error)
			dlgcall, _ := a[1].(func(...interface{}) error)
			switch lgtpe {
			case LogInfo:
				if dlgcall != nil && infocall == nil {
					infocall = dlgcall
				} else if dlgfcall != nil && infofcall == nil {
					infofcall = dlgfcall
				}
			case LogWarning:
				if dlgcall != nil && warncall == nil {
					warncall = dlgcall
				} else if dlgfcall != nil && warnfcall == nil {
					warnfcall = dlgfcall
				}
			case LogError:
				if dlgcall != nil && errorcall == nil {
					errorcall = dlgcall
				} else if dlgfcall != nil && errorfcall == nil {
					errorfcall = dlgfcall
				}
			default:
				a = nil
			}
			if len(a) >= 2 && len(a)%2 == 0 {
				a = a[2:]
			} else {
				break
			}
		} else {
			break
		}
	}

	if infocall == nil {
		infocall = func(a ...interface{}) error {
			return lggngwrp.Info(a...)
		}
	}
	if infofcall == nil {
		infofcall = func(format string, a ...interface{}) error {
			return lggngwrp.Infof(format, a...)
		}
	}

	if warncall == nil {
		warncall = func(a ...interface{}) error {
			return lggngwrp.Warning(a...)
		}
	}
	if warnfcall == nil {
		warnfcall = func(format string, a ...interface{}) error {
			return lggngwrp.Warningf(format, a...)
		}
	}

	if errorcall == nil {
		errorcall = func(a ...interface{}) error {
			return lggngwrp.Error(a...)
		}
	}
	if errorfcall == nil {
		errorfcall = func(format string, a ...interface{}) error {
			return lggngwrp.Errorf(format, a...)
		}
	}
	lggng = &logginghandler{activecall: func() (active bool) {
		return lggngwrp.Active()
	}, infocall: infocall, infofcall: infofcall, warncall: warncall, warnfcall: warnfcall, errorcall: errorcall, errorfcall: errorfcall}
	return
}

func (lggngwrp *loggingwrap) Info(v ...interface{}) (err error) {
	if lggngwrp != nil && lggngwrp.Active() && lggngwrp.lggng != nil {
		err = lggngwrp.lggng.log(LogInfo, v...)
	}
	return
}

func (lggngwrp *loggingwrap) Errorf(format string, v ...interface{}) (err error) {
	if lggngwrp != nil && lggngwrp.Active() && lggngwrp.lggng != nil {
		err = lggngwrp.lggng.logf(LogError, format, v...)
	}
	return
}

func (lggngwrp *loggingwrap) Warningf(format string, v ...interface{}) (err error) {
	if lggngwrp != nil && lggngwrp.Active() && lggngwrp.lggng != nil {
		err = lggngwrp.lggng.logf(LogWarning, format, v...)
	}
	return
}

func (lggngwrp *loggingwrap) Infof(format string, v ...interface{}) (err error) {
	if lggngwrp != nil && lggngwrp.Active() && lggngwrp.lggng != nil {
		err = lggngwrp.lggng.logf(LogInfo, format, v...)
	}
	return
}

type logging struct {
	FS         *fsutils.FSUtils
	logname    string
	logpath    string
	localpath  string
	validtypes map[logtype]bool
}

func logtheentry(FS *fsutils.FSUtils,
	logname string,
	logpath string,
	localpath string,
	lgtpe logtype,
	timestamp time.Time,
	args ...interface{}) {
	timestamps := strings.Split(timestamp.Format(time.RFC3339), "T")
	timestampyear := timestamps[0]
	timestamptimes := strings.Split(timestamps[1], "+")
	timestamtime := timestamptimes[0]
	timestampmilsecs := strconv.FormatInt(int64(timestamp.Nanosecond()), 10)
	if len(timestampmilsecs) > 6 {
		timestampmilsecs = timestampmilsecs[:6]
	}
	logfilepath := logpath + func() string {
		if !strings.HasSuffix(logpath, "/") {
			return "/"
		} else {
			return ""
		}
	}() + logname + "." + strings.Replace(strings.Split(timestamp.Format(time.RFC3339), "T")[0], "-", "", -1) + ".log"
	FS.APPEND(logfilepath, append(append([]interface{}{lgtpe.String() + ":", "[", timestampyear, " ", timestamtime, ".", timestampmilsecs, "] "}, args), "\r\n")...)
}

func (lggng *logging) logf(lgtype logtype, format string, a ...interface{}) (err error) {
	err = lggng.log(lgtype, fmt.Sprintf(format, a...))
	return
}

func (lggng *logging) log(lgtype logtype, a ...interface{}) (err error) {
	if lggng != nil && lggng.FS != nil && lggng.validtypes[lgtype] {
		go logtheentry(lggng.FS, lggng.logname, lggng.logpath, lggng.localpath, lgtype, time.Now(), a...)
	}
	return
}

type LoggingManager struct {
	lggrs    map[string]*logging
	rsngmngr *resources.ResourcingManager
}

type logtype int

const (
	LogUnknown logtype = iota
	LogInfo
	LogWarning
	LogError
)

func (lgtpe logtype) String() (s string) {
	switch lgtpe {
	case LogInfo:
		s = "INFO"
	case LogWarning:
		s = "WARNING"
	case LogError:
		s = "ERROR"
	default:
		s = "UNKNOWN"
	}
	return
}

func (lggngmngr *LoggingManager) Register(alias string, path ...interface{}) (logger Logger, err error) {
	if lggngmngr != nil && alias != "" {
		if _, ok := lggngmngr.lggrs[alias]; !ok {
			validtypes := []logtype{}
			pathi := 0
			pathl := len(path)
			var fs *fsutils.FSUtils = nil
			for pathi < pathl {
				if dlgtpe, dlgtpeok := path[pathi].(logtype); dlgtpeok {
					path = append(path[:pathi], path[pathi+1:]...)
					pathl--
					validtypes = append(validtypes, dlgtpe)
					continue
				} else if dfs, dfsok := path[pathi].(*fsutils.FSUtils); dfsok {
					if dfs != nil {
						if fs == nil {
							fs = dfs
						}
					}
					path = append(path[:pathi], path[pathi+1:]...)
					pathl--
				}
				pathi++
			}
			if fs == nil {
				fs = lggngmngr.rsngmngr.FS()
			}
			if fs.MKDIR(path...) {
				if len(validtypes) == 0 {
					validtypes = []logtype{LogInfo, LogError, LogWarning}
				}
				lggr := &logging{FS: fs, logname: alias, validtypes: map[logtype]bool{}, logpath: fmt.Sprint(path[0]), localpath: fmt.Sprint(path[1])}

				for _, lgtpe := range validtypes {
					lggr.validtypes[lgtpe] = true
				}
				if lggngmngr.lggrs == nil {
					lggngmngr.lggrs = map[string]*logging{}
				}
				lggngmngr.lggrs[alias] = lggr
				logger = &loggingwrap{lggng: lggr}
			}
		}
	}
	return
}

func (lggngmngr *LoggingManager) Logger(alias string) (logger Logger) {
	if lggngmngr != nil && alias != "" {
		lggr := lggngmngr.lggrs[alias]
		logger = &loggingwrap{lggng: lggr}
	}
	return
}

func (lggngmngr *LoggingManager) Error(alias string, v ...interface{}) (err error) {
	err = lggngmngr.Log(alias, LogError, v...)
	return
}

func (lggngmngr *LoggingManager) Warning(alias string, v ...interface{}) (err error) {
	err = lggngmngr.Log(alias, LogWarning, v...)
	return
}

func (lggngmngr *LoggingManager) Info(alias string, v ...interface{}) (err error) {
	err = lggngmngr.Log(alias, LogInfo, v...)
	return
}

func (lggngmngr *LoggingManager) Errorf(alias string, format string, v ...interface{}) (err error) {
	err = lggngmngr.Logf(alias, LogError, format, v...)
	return
}

func (lggngmngr *LoggingManager) Warningf(alias string, format string, v ...interface{}) (err error) {
	err = lggngmngr.Logf(alias, LogWarning, format, v...)
	return
}

func (lggngmngr *LoggingManager) Infof(alias string, format string, v ...interface{}) (err error) {
	err = lggngmngr.Logf(alias, LogInfo, format, v...)
	return
}

func (lggngmngr *LoggingManager) Logf(alias string, lgtype logtype, format string, v ...interface{}) (err error) {
	if lggngmngr != nil {
		lggngmngr.Log(alias, lgtype, fmt.Sprintf(format, v...))
	}
	return
}

func (lggngmngr *LoggingManager) FS() (fs *fsutils.FSUtils) {
	if lggngmngr != nil && lggngmngr.rsngmngr != nil {
		fs = lggngmngr.rsngmngr.FS()
	}
	return
}

func (lggngmngr *LoggingManager) Log(alias string, lgtype logtype, v ...interface{}) (err error) {
	if lggngmngr != nil && alias != "" {
		if lggng := lggngmngr.lggrs[alias]; lggng != nil {
			err = lggng.log(lgtype, v...)
		}
	}
	return
}

var glblgngmngr *LoggingManager = &LoggingManager{rsngmngr: resources.GLOBALRSNG()}

func GLOBALLOGGING() *LoggingManager {
	return glblgngmngr
}

func DummyLogger() Logger {
	return &loggingwrap{lggng: nil}
}
