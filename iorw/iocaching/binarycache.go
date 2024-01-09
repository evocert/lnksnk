package iocaching

import (
	"sync"
	"time"

	"github.com/evocert/lnksnk/iorw"
)

type BufferCache struct {
	cachbuffs  *sync.Map
	cachstamps *sync.Map
}

func NewBufferCache() (buffcch *BufferCache) {
	buffcch = &BufferCache{cachbuffs: &sync.Map{}, cachstamps: &sync.Map{}}
	return
}

func (buffcch *BufferCache) Set(alias string, a ...interface{}) {
	if al := len(a); al > 0 && buffcch != nil && alias != "" {
		if cachbuffs, cachstamps := buffcch.cachbuffs, buffcch.cachstamps; cachbuffs != nil && cachstamps != nil {
			var modfied = time.Now()
			for an := 0; an < al; {
				if modd, moddok := a[an].(time.Time); moddok {
					modfied = modd
					a = append(a[:an], a[an+1:]...)
					al--
					continue
				}
				an++
			}
			if loadedbuffv, loaded := cachbuffs.Load(alias); !loaded {
				buff := iorw.NewBuffer()
				buff.Print(a...)
				cachbuffs.Store(alias, buff)
				cachstamps.Store(alias, modfied)
			} else if loaded {
				buff := loadedbuffv.(*iorw.Buffer)
				modbyv, modnyok := cachstamps.Load(alias)
				if modnyok && modfied != modbyv.(time.Time) {
					buff.Clear()
					buff.Print(a...)
					cachstamps.Store(alias, modfied)
				}
			}
		}
	}
}

func (buffcch *BufferCache) Del(alias string) {
	if buffcch != nil {
		if cachbuffs, cachstamps := buffcch.cachbuffs, buffcch.cachstamps; cachbuffs != nil && cachstamps != nil {
			if prvbuf, prvbufok := cachbuffs.Load(alias); prvbufok && (prvbuf == nil || prvbuf != nil) {
				_, prvbufok = cachbuffs.LoadAndDelete(alias)
				if prvbufok {
					cachstamps.Delete(alias)
					if prvbuf != nil {
						if buf, _ := prvbuf.(*iorw.Buffer); buf != nil {
							buf.Close()
						}
					}
				}
			}
		}
	}
}

func (buffcch *BufferCache) ReaderReplace(alias string, testmodified time.Time, a ...interface{}) (buffrdr *iorw.BuffReader) {
	if al := len(a); al > 0 && buffcch != nil && alias != "" {
		if bufrdr, buffmod := buffcch.Reader(alias); bufrdr == nil || (bufrdr != nil && testmodified != buffmod) {
			buffcch.Set(alias, append(a, testmodified)...)
			buffrdr, _ = buffcch.Reader(alias)
		} else if bufrdr != nil {
			buffrdr = bufrdr
		}
	}
	return
}

func (buffcch *BufferCache) Reader(alias string) (buffrdr *iorw.BuffReader, lastmodified time.Time) {
	if buffcch != nil && alias != "" {
		if cachbuffs, cachstamps := buffcch.cachbuffs, buffcch.cachstamps; cachbuffs != nil && cachstamps != nil {
			if buff, buffok := cachbuffs.Load(alias); buffok {
				buffrdr = buff.(*iorw.Buffer).Reader()
				if modv, modok := cachstamps.Load(alias); modok {
					lastmodified = modv.(time.Time)
				}
			}
		}
	}
	return
}
