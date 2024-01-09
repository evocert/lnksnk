package serve

import (
	"bufio"
	"net/http"
	"strconv"
	"strings"
)

type reader struct {
	httpr       *http.Request
	bufr        *bufio.Reader
	rangetype   string
	rangeoffset int64
}

func (rqr *reader) RangeOffset() int64 {
	if rqr != nil {
		return rqr.rangeoffset
	}
	return -1
}

func (rqr *reader) RangeType() string {
	if rqr != nil {
		return rqr.rangetype
	}
	return ""
}

func (rqr *reader) buffer() (bufr *bufio.Reader) {
	if rqr != nil {
		if bufr = rqr.bufr; bufr == nil {
			if httpr := rqr.httpr; httpr != nil {
				if r := httpr.Body; r != nil {
					bufr = bufio.NewReaderSize(r, 65536)
				}
			}
		}
	}
	return
}

func (rqr *reader) Read(p []byte) (n int, err error) {
	if rqr != nil {
		if bufr := rqr.buffer(); bufr != nil {
			n, err = bufr.Read(p)
		}
	}
	return
}

func (rqr *reader) ReadRune() (r rune, size int, err error) {
	if rqr != nil {
		if bufr := rqr.buffer(); bufr != nil {
			r, size, err = bufr.ReadRune()
		}
	}
	return
}

func (rqr *reader) Close() (err error) {
	if rqr != nil {
		if rqr.httpr != nil {
			rqr.httpr = nil
		}
		if rqr.bufr != nil {
			rqr.bufr = nil
		}
	}
	return
}

func newReader(httpr *http.Request) (rdr *reader) {
	rdr = &reader{httpr: httpr, rangeoffset: -1}
	if httpr != nil {
		prtclrangetype := ""
		prtclrangeoffset := int64(-1)
		if prtclrange := httpr.Header.Get("Range"); prtclrange != "" && strings.Index(prtclrange, "=") > 0 {
			if prtclrangetype = prtclrange[:strings.Index(prtclrange, "=")]; prtclrange != "" {
				if prtclrange = prtclrange[strings.Index(prtclrange, "=")+1:]; strings.Index(prtclrange, "-") > 0 {
					prtclrangeoffset, _ = strconv.ParseInt(prtclrange[:strings.Index(prtclrange, "-")], 10, 64)
				}
			}
		}
		rdr.rangeoffset = prtclrangeoffset
		rdr.rangetype = prtclrangetype
	}
	return
}
