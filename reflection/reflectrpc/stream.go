package reflectrpc

import (
	"bufio"
	"io"

	"github.com/evocert/lnksnk/parameters"
)

type rpcstream struct {
	conn   io.ReadWriteCloser
	rdr    io.ReadCloser
	bfrw   *bufio.ReadWriter
	wtr    io.WriteCloser
	rpcitm *rpcitem
	params parameters.ParametersAPI
}

func NewRpcStream(rpcitm *rpcitem, conn io.ReadWriteCloser, rdr io.ReadCloser, wtr io.WriteCloser, bfrw *bufio.ReadWriter, params parameters.ParametersAPI) (rpcstrm *rpcstream) {
	if rpcitm != nil {
		if conn != nil && rdr == nil {
			wtr, _ = conn.(io.WriteCloser)
			rdr, _ = conn.(io.ReadCloser)
			if rdr != nil && wtr != nil && bfrw != nil {
				bfrw = nil
			}
			rpcstrm = &rpcstream{rpcitm: rpcitm, params: params, conn: conn, rdr: rdr, wtr: wtr, bfrw: bfrw}
		} else if rdr != nil && wtr != nil {
			if rdr != nil && wtr != nil && bfrw != nil {
				bfrw = nil
			}
			rpcstrm = &rpcstream{rpcitm: rpcitm, params: params, conn: conn, rdr: rdr, wtr: wtr, bfrw: bfrw}
		}
	}
	return
}

func (rpcstrm *rpcstream) ReadRune() (r rune, size int, err error) {
	if rpcstrm != nil {
		if bfrw, rdr, wtr := rpcstrm.bfrw, rpcstrm.rdr, rpcstrm.wtr; rdr != nil && wtr != nil {
			if bfrw == nil {
				bfrw = bufio.NewReadWriter(bufio.NewReaderSize(rdr, 4096), bufio.NewWriterSize(wtr, 4096))
			}
			r, size, err = bfrw.ReadRune()
		}
	}
	return
}

func (rpcstrm *rpcstream) WriteRune(r rune) (size int, err error) {
	if rpcstrm != nil {
		if bfrw, rdr, wtr := rpcstrm.bfrw, rpcstrm.rdr, rpcstrm.wtr; rdr != nil && wtr != nil {
			if bfrw == nil {
				bfrw = bufio.NewReadWriter(bufio.NewReaderSize(rdr, 4096), bufio.NewWriterSize(wtr, 4096))
			}
			size, err = bfrw.WriteRune(r)
		}
	}
	return
}

func (rpcstrm *rpcstream) Read(p []byte) (n int, err error) {
	if rpcstrm != nil {
		if bfrw, rdr, wtr := rpcstrm.bfrw, rpcstrm.rdr, rpcstrm.wtr; rdr != nil && wtr != nil {
			if bfrw == nil {
				bfrw = bufio.NewReadWriter(bufio.NewReaderSize(rdr, 4096), bufio.NewWriterSize(wtr, 4096))
			}
			n, err = bfrw.Read(p)
		}
	}
	return
}

func (rpcstrm *rpcstream) Write(p []byte) (n int, err error) {
	if rpcstrm != nil {

	}
	return
}

func (rpcstrm *rpcstream) Close() (err error) {
	if rpcstrm != nil {

	}
	return
}
