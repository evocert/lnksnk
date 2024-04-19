package iorw

import "io"

type RuneReaderSlice struct {
	rnrdrs  []io.RuneReader
	crntrdr io.RuneReader
}

func NewRuneReaderSlice(rnrdrs ...io.RuneReader) (rnrdrsslce *RuneReaderSlice) {
	rnrdrsslce = &RuneReaderSlice{crntrdr: nil, rnrdrs: append([]io.RuneReader{}, rnrdrs...)}
	return
}

func (rnrdrsslce *RuneReaderSlice) Length() (ln int) {
	if rnrdrsslce != nil {
		ln = len(rnrdrsslce.rnrdrs)
	}
	return
}

func (rnrdrsslce *RuneReaderSlice) PreAppend(rdrs ...io.RuneReader) {
	if rnrdrsslce != nil {
		if len(rdrs) > 0 {
			if rnrdrsslce.crntrdr != nil {
				rdrs = append(rdrs, rnrdrsslce.crntrdr)
				rnrdrsslce.crntrdr = nil
			}
			rnrdrsslce.rnrdrs = append(rdrs, rnrdrsslce.rnrdrs...)
		}
	}
}

func (rnrdrsslce *RuneReaderSlice) PostAppend(rdrs ...io.RuneReader) {
	if rnrdrsslce != nil {
		if len(rdrs) > 0 {
			rnrdrsslce.rnrdrs = append(rnrdrsslce.rnrdrs, rdrs...)
		}
	}
}

func (rnrdrsslce *RuneReaderSlice) ReadRune() (r rune, size int, err error) {
	if rnrdrsslce != nil {
		if rnrdrsslce.crntrdr != nil {
			r, size, err = rnrdrsslce.crntrdr.ReadRune()
			if (err == io.EOF && size == 0) || size == 0 {
				rnrdrsslce.crntrdr = nil
				r, size, err = rnrdrsslce.ReadRune()
			}
		} else if rdrsl := len(rnrdrsslce.rnrdrs); rnrdrsslce.crntrdr == nil {
			if rdrsl > 0 {
				rnrdrsslce.crntrdr = rnrdrsslce.rnrdrs[0]
				rnrdrsslce.rnrdrs = rnrdrsslce.rnrdrs[1:]
				r, size, err = rnrdrsslce.ReadRune()
			} else {
				err = io.EOF
			}
		}
	}

	return
}

func (rnrdrsslce *RuneReaderSlice) Close() (err error) {
	if rnrdrsslce != nil {
		if rnrdrsl := len(rnrdrsslce.rnrdrs); rnrdrsl > 0 {
			for rnrdrsl > 0 {
				rnrdrsslce.rnrdrs[0] = nil
				rnrdrsslce.rnrdrs = rnrdrsslce.rnrdrs[1:]
				rnrdrsl--
			}
			rnrdrsslce.rnrdrs = nil
		}
		if rnrdrsslce.crntrdr != nil {
			rnrdrsslce.crntrdr = nil
		}
	}
	return
}
