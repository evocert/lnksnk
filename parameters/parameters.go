package parameters

import (
	"bufio"
	"io"
	"mime/multipart"
	http "net/http"
	url "net/url"
	"strings"
	"sync"
)

type ParametersAPI interface {
	StandardKeys() []string
	FileKeys() []string
	SetParameter(string, bool, ...string)
	ContainsParameters() bool
	AppendPhrase(...string)
	Phrases() []string
	ContainsPhrase(...string) bool
	ContainsParameter(string) bool
	RemoveParameter(string) []string
	SetFileParameter(string, bool, ...interface{})
	ContainsFileParameter(string) bool
	Parameter(string, ...int) []string
	StringParameter(string, string, ...int) string
	FileReader(string, ...int) []io.Reader
	FileName(string, ...int) []string
	FileSize(string, ...int) []int64
	FileParameter(string, ...int) []interface{}
	CleanupParameters()
	Type(string) string
}

// Parameters -> structure containing parameters
type Parameters struct {
	phrases        []string
	urlkeys        *sync.Map
	standard       *sync.Map //map[string][]string
	standardcount  int
	filesdata      *sync.Map //map[string][]interface{}
	filesdatacount int
}

var emptyParmVal = []string{}
var emptyParamFile = []interface{}{}

func (params *Parameters) AppendPhrase(phrases ...string) {
	if params != nil {
		if len(phrases) > 0 {
			for phrsn := range phrases {
				if phrs := phrases[phrsn]; phrs != "" {
					if params.phrases == nil {
						params.phrases = []string{}
					}
					params.phrases = append(params.phrases, phrs)
				}
			}
		}
	}
}

func (params *Parameters) Phrases() (phrases []string) {
	if params != nil && len(params.phrases) > 0 {
		phrases = params.phrases[:]
	}
	return
}

func (params *Parameters) ContainsPhrase(...string) (exists bool) {

	return
}

// StandardKeys - list of standard parameters names (keys)
func (params *Parameters) StandardKeys() (keys []string) {
	if params != nil {
		if standard, standardcount := params.standard, params.standardcount; standard != nil && standardcount > 0 {
			if keys == nil {
				keys = make([]string, standardcount)
			}
			ki := 0
			standard.Range(func(key, value any) bool {
				if ki < standardcount {
					keys[ki] = key.(string)
					ki++
				}
				return true
			})
		}
	}
	return keys
}

// FileKeys - list of file parameters names (keys)
func (params *Parameters) FileKeys() (keys []string) {
	if params != nil {
		if filesdata, filesdatacount := params.filesdata, params.filesdatacount; filesdata != nil && filesdatacount > 0 {
			if keys == nil {
				keys = make([]string, filesdatacount)
			}
			ki := 0
			filesdata.Range(func(key, value any) bool {
				if ki < filesdatacount {
					keys[ki] = key.(string)
					ki++
				}
				return true
			})
		}
	}
	return keys
}

// SetParameter -> set or append parameter value
// pname : name
// pvalue : value of strings to add
// clear : clear existing value of parameter
func (params *Parameters) SetParameter(pname string, clear bool, pvalue ...string) {
	storeParameter(params, false, pname, clear, pvalue...)
}

func storeParameter(params *Parameters, isurl bool, pname string, clear bool, pvalue ...string) {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return
	}

	if params != nil {
		standard, urlkeys := params.standard, params.urlkeys
		if urlkeys == nil {
			urlkeys = &sync.Map{}
			params.urlkeys = urlkeys
		}
		if standard == nil {
			standard = &sync.Map{} // make(map[string][]string)
			params.standard = standard
		}
		if val, ok := standard.Load(pname); ok {
			if clear {
				standard.Swap(pname, []string{})
				val, _ = standard.Load(pname)
				urlkeys.LoadAndDelete(pname)
			}
			var valsarr, _ = val.([]string)
			if len(pvalue) > 0 {
				valsarr = append(valsarr, pvalue...)
				urlkeys.Store(pname, isurl)
			}
			params.standard.Swap(pname, valsarr)
		} else {
			if len(pvalue) > 0 {
				urlkeys.Store(pname, isurl)
				params.standard.Store(pname, pvalue[:])
				params.standardcount++
			} else {
				params.standard.Store(pname, []string{})
				params.standardcount++
			}
		}
	}
}

// ContainsParameter -> check if parameter exist
// pname : name
func (params *Parameters) ContainsParameter(pname string) bool {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return false
	}
	standard := params.standard
	if standard == nil {
		return false
	}
	_, ok := standard.Load(pname)
	return ok
}

// Type -> check if parameter was loaded as a url/standard parameter
// pname : name
func (params *Parameters) Type(pname string) string {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return ""
	}
	standard, urlkeys := params.standard, params.urlkeys
	if urlkeys == nil && standard == nil {
		return ""
	}
	if isurlv, ok := urlkeys.Load(pname); ok {
		if ok, _ = isurlv.(bool); ok {
			return "url"
		}
	}
	if _, ok := standard.Load(pname); ok {
		return "std"
	}
	return ""
}

// RemoveParameter  -> remove parameter and return any slice of string value
func (params *Parameters) RemoveParameter(pname string) (value []string) {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return
	}
	standard, urlkeys := params.standard, params.urlkeys
	if standard == nil {
		return
	}
	if stdval, ok := standard.LoadAndDelete(pname); ok {
		if urlkeys != nil {
			urlkeys.LoadAndDelete(pname)
		}
		params.standardcount--
		value, _ = stdval.([]string)
	}
	return
}

// ContainsParameters  -> return true if there are parameters
func (params *Parameters) ContainsParameters() (contains bool) {
	if params != nil {
		if standard := params.standard; standard != nil {
			standard.Range(func(key, value any) bool {
				contains = true
				return !contains
			})
		}
	}
	return
}

// SetFileParameter -> set or append file parameter value
// pname : name
// pfile : value of interface to add either FileHeader from mime/multipart or any io.Reader implementation
// clear : clear existing value of parameter
func (params *Parameters) SetFileParameter(pname string, clear bool, pfile ...interface{}) {
	if params != nil {
		if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
			return
		}
		filesdata := params.filesdata
		if filesdata == nil {
			filesdata = &sync.Map{}
			params.filesdata = filesdata
		}
		if fval, ok := filesdata.Load(pname); ok {
			var val, _ = fval.([]interface{})
			if clear {
				val = []interface{}{}
				filesdata.Store(pname, val)
			}
			if len(pfile) > 0 {
				//for pf := range pfile {
				//	val = append(val, pfile[pf])
				//}
				val = append(val, pfile...)
			}
			filesdata.Store(pname, val)
		} else {
			if len(pfile) > 0 {
				var val = []interface{}{}
				for pf := range pfile {
					if fheader, fheaderok := pfile[pf].(multipart.FileHeader); fheaderok {
						if fv, fverr := fheader.Open(); fverr == nil {
							if rval, rvalok := fv.(io.Reader); rvalok {
								val = append(val, rval)
							}
						}
					} else {
						val = append(val, pfile[pf])
					}
				}
				filesdata.Store(pname, val)
				params.filesdatacount++
			} else {
				filesdata.Store(pname, []interface{}{})
				params.filesdatacount++
			}
		}
	}
}

// ContainsFileParameters -> return true if file parameters exist
func (params *Parameters) ContainsFileParameters() (contains bool) {
	if params != nil {
		filesdata := params.filesdata
		if filesdata == nil {
			return
		}
		filesdata.Range(func(key, value any) bool {
			contains = true
			return !contains
		})
	}
	return
}

// ContainsFileParameter -> check if file parameter exist
// pname : name
func (params *Parameters) ContainsFileParameter(pname string) bool {
	if params != nil {
		if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
			return false
		}
		filesdata := params.filesdata
		if filesdata == nil {
			return false
		}
		_, ok := filesdata.Load(pname)
		return ok
	}
	return false
}

// Parameter - return a specific parameter values
func (params *Parameters) Parameter(pname string, index ...int) []string {
	if params != nil {
		if pname = strings.ToUpper(strings.TrimSpace(pname)); pname != "" {
			if standard := params.standard; standard != nil {
				if stdval, ok := standard.Load(pname); ok {
					stdv, _ := stdval.([]string)
					if stdl := len(stdv); stdl > 0 {
						if il := len(index); il > 0 {
							idx := []int{}
							for idn := range index {
								if id := index[idn]; id >= 0 && id < stdl {
									idx = append(idx, id)
								}
							}
							if len(idx) > 0 {
								stdvls := make([]string, len(idx))
								for in := range idx {
									stdvls[in] = stdv[idx[in]]
								}
								return stdvls
							}
						} else {
							return stdv
						}
					}
				}
			}
		}
	}
	return emptyParmVal
}

// StringParameter return parameter as string concatenated with sep
func (params *Parameters) StringParameter(pname string, sep string, index ...int) (s string) {
	if params != nil {
		if pval := params.Parameter(pname, index...); len(pval) > 0 {
			return strings.Join(pval, sep)
		}
		if pval := params.FileReader(pname, index...); len(pval) > 0 {
			var rnrtos = func(br *bufio.Reader) (bs string, err error) {
				rns := make([]rune, 1024)
				rnsi := 0
				if br != nil {
					for {
						rn, size, rnerr := br.ReadRune()
						if size > 0 {
							rns[rnsi] = rn
							rnsi++
							if rnsi == len(rns) {
								bs += string(rns[:rnsi])
								rnsi = 0
							}
						}
						if rnerr != nil {
							if rnerr != io.EOF {
								err = rnerr
							}
							break
						}
					}
				}
				if rnsi > 0 {
					bs += string(rns[:rnsi])
					rnsi = 0
				}
				return
			}
			var bfr *bufio.Reader = nil
			for rn := range pval {
				if r := pval[rn]; r != nil {
					if bfr == nil {
						bfr = bufio.NewReader(r)
					} else {
						bfr.Reset(r)
					}
					if bfr != nil {
						if bs, bserr := rnrtos(bfr); bserr == nil {
							s += bs
						} else {
							break
						}
					}
				}
				if rn < len(pval)-1 {
					s += sep
				}
			}
		}
	}
	return
}

// FileReader return file parameter - array of io.Reader
func (params *Parameters) FileReader(pname string, index ...int) (rdrs []io.Reader) {
	if params != nil {
		if flsv := params.FileParameter(pname, index...); len(flsv) > 0 {
			rdrs = make([]io.Reader, len(flsv))
			for nfls, fls := range flsv {
				if fhead, fheadok := fls.(*multipart.FileHeader); fheadok {
					rdrs[nfls], _ = fhead.Open()
				} else if fr, frok := fls.(io.Reader); frok {
					rdrs[nfls] = fr
				}
			}
		}
	}
	return
}

// FileName return file parameter name - array of string
func (params *Parameters) FileName(pname string, index ...int) (nmes []string) {
	if params != nil {
		if flsv := params.FileParameter(pname, index...); len(flsv) > 0 {
			nmes = make([]string, len(flsv))
			for nfls := range flsv {
				if fhead, fheadok := flsv[nfls].(*multipart.FileHeader); fheadok {
					nmes[nfls] = fhead.Filename
				}
			}
		}
	}
	return
}

// FileSize return file parameter size - array of int64)
func (params *Parameters) FileSize(pname string, index ...int) (sizes []int64) {
	if params != nil {
		if flsv := params.FileParameter(pname, index...); len(flsv) > 0 {
			sizes = make([]int64, len(flsv))
			for nfls, fls := range flsv {
				if fhead, fheadok := fls.(multipart.FileHeader); fheadok {
					sizes[nfls] = fhead.Size
				}
			}
		}
	}
	return
}

// FileParameter return file paramater - array of file
func (params *Parameters) FileParameter(pname string, index ...int) []interface{} {
	if params != nil {
		if pname = strings.ToUpper(strings.TrimSpace(pname)); pname != "" {
			filesdata := params.filesdata
			if filesdata != nil {
				if flsvv, ok := filesdata.Load(pname); ok {
					var flsv, _ = flsvv.([]interface{})
					if flsl := len(flsv); flsl > 0 {
						if il := len(index); il > 0 {
							idx := []int{}
							for _, id := range index {
								if id >= 0 && id < flsl {
									idx = append(idx, id)
								}
							}
							if len(idx) > 0 {
								flsvls := make([]interface{}, len(idx))
								for in, id := range idx {
									flsvls[in] = flsv[id]
								}
								return flsvls
							}
						} else {
							return flsv
						}
					}
				}
			}
		}
	}
	return emptyParamFile
}

// CleanupParameters function that can be called to assist in cleaning up instance of Parameter container
func (params *Parameters) CleanupParameters() {
	if standard, urlkeys := params.standard, params.urlkeys; standard != nil {
		params.standard = nil
		params.urlkeys = nil
		params.standardcount = 0
		standard.Range(func(key, value any) bool {
			if urlkeys != nil {
				urlkeys.LoadAndDelete(key)
			}
			_, delok := standard.LoadAndDelete(key)
			return !delok
		})
	}
	if filesdata := params.filesdata; filesdata != nil {
		params.filesdata = nil
		params.filesdatacount = 0
		filesdata.Range(func(key, value any) bool {
			_, delok := filesdata.LoadAndDelete(key)
			return !delok
		})
	}
	if params != nil {
		if phrsl := len(params.phrases); phrsl > 0 {
			for phrsl > 0 {
				params.phrases[0] = ""
				params.phrases = params.phrases[1:]
				phrsl--
			}
			params.phrases = nil
		}
	}
}

// NewParameters return new instance of Paramaters container
func NewParameters() *Parameters {
	return &Parameters{}
}

// LoadParametersFromRawURL - populate paramaters just from raw url
func LoadParametersFromRawURL(params ParametersAPI, rawURL string) {
	if params != nil && rawURL != "" {
		if rawURL != "" {
			rawURL = strings.Replace(rawURL, ";", "%3b", -1)
			var phrases = []string{}
			var rawUrls = strings.Split(rawURL, "&")
			rawURL = ""
			for _, rwurl := range rawUrls {
				if rwurl != "" {
					if strings.Contains(rwurl, "=") {
						rawURL += rwurl + "&"
						continue
					}
					phrases = append(phrases, rwurl)
					continue
				}
			}
			if len(rawURL) > 1 && strings.HasSuffix(rawURL, "&") {
				rawURL = rawURL[:len(rawURL)-1]
			}
			if urlvals, e := url.ParseQuery(rawURL); e == nil {
				if len(urlvals) > 0 {
					for pname, pvalue := range urlvals {
						storeParameter(params.(*Parameters), true, pname, false, pvalue...)
					}
				}
			}
			if len(phrases) > 0 {
				params.AppendPhrase(phrases...)
			}
		}
	}
}

// LoadParametersFromUrlValues - Load Parameters from url.Values
func LoadParametersFromUrlValues(params ParametersAPI, urlvalues url.Values) (err error) {
	if params != nil && urlvalues != nil {
		for pname, pvalue := range urlvalues {
			params.SetParameter(pname, false, pvalue...)
		}
	}
	return
}

// LoadParametersFromMultipartForm - Load Parameters from *multipart.Form
func LoadParametersFromMultipartForm(params ParametersAPI, mpartform *multipart.Form) (err error) {
	if params != nil && mpartform != nil {
		for pname, pvalue := range mpartform.Value {
			params.SetParameter(pname, false, pvalue...)
		}
		for pname, pfile := range mpartform.File {
			if len(pfile) > 0 {
				pfilei := []interface{}{}
				for _, pf := range pfile {
					pfilei = append(pfilei, pf)
				}
				params.SetFileParameter(pname, false, pfilei...)
				pfilei = nil
			}
		}
	}
	return
}

// LoadParametersFromHTTPRequest - Load Parameters from http.Request
func LoadParametersFromHTTPRequest(params ParametersAPI, r *http.Request) {
	if params != nil {
		if r.URL != nil {
			LoadParametersFromRawURL(params, r.URL.RawQuery)
			r.URL.RawQuery = ""
		}
		if err := r.ParseMultipartForm(0); err == nil {
			if r.MultipartForm != nil {
				LoadParametersFromMultipartForm(params, r.MultipartForm)
			} else if r.Form != nil {
				LoadParametersFromUrlValues(params, r.Form)
			}
		} else if err := r.ParseForm(); err == nil {
			LoadParametersFromUrlValues(params, r.Form)
		}
	}
}
