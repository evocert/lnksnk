package resources

import "io"

//Resource - struct
type Resource struct {
	rsngepnt *ResourcingEndpoint
	rspath   string
	isText   bool
	isBin    bool
	r        io.Reader
	rs       io.Seeker
	rc       io.Closer
}

func newRS(rsngepnt *ResourcingEndpoint, rspath string, r io.Reader) (rs *Resource) {
	if r != nil {
		rs = &Resource{rsngepnt: rsngepnt, rspath: rspath, r: r}
		if sr, srok := r.(io.Seeker); srok {
			rs.rs = sr
		}
		if rc, rcok := r.(io.Closer); rcok {
			rs.rc = rc
		}
	}
	return
}

//Seek - refer to io.Seeker
func (rs *Resource) Seek(offset int64, whence int) (n int64, err error) {
	if rs != nil && rs.rs != nil {
		n, err = rs.rs.Seek(offset, whence)
	}
	return
}

func (rs *Resource) Read(p []byte) (n int, err error) {
	if rs != nil && rs.r != nil {
		if pl := len(p); pl > 0 {
			n, err = rs.r.Read(p)
		}
	}
	return
}

//Close - refer to io.Closer
func (rs *Resource) Close() (err error) {
	if rs != nil {
		if rs.r != nil {
			rs.r = nil
		}
		if rs.rs != nil {
			rs.rs = nil
		}
		if rs.rc != nil {
			err = rs.rc.Close()
			rs.rc = nil
		}
		if rs.rsngepnt != nil {
			rs.rsngepnt = nil
		}
		rs = nil
	}
	return
}
