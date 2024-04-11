package iorw

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"unicode/utf8"
)

// Buffer -
type Buffer struct {
	insertedbuffs map[int64]*Buffer
	buffer        [][]byte
	bytes         []byte
	bytesi        int
	lck           *sync.RWMutex
	bufrs         map[*BuffReader]*BuffReader
	OnClose       func(*Buffer)
	MaxLenToWrite int64
	maxwrttnl     int64
	OnMaxWritten  func(int64) bool
}

// NewBuffer -
func NewBuffer(a ...interface{}) (buff *Buffer) {
	buff = &Buffer{lck: &sync.RWMutex{}, maxwrttnl: -1, MaxLenToWrite: -1, buffer: [][]byte{}, bytesi: 0, bytes: make([]byte, 8192), bufrs: map[*BuffReader]*BuffReader{}, insertedbuffs: map[int64]*Buffer{}}
	if len(a) > 0 {
		buff.Print(a...)
	}
	return
}

func (buff *Buffer) InsertAt(offset int64, whence int, a ...interface{}) (err error) {
	if buff != nil {
		if al := len(a); al > 0 && offset > -1 {
			if whence == io.SeekStart || whence == io.SeekCurrent || whence == io.SeekEnd {
				func() {
					buff.lck.RLock()
					if s := buff.Size(); offset < s {
						if whence == io.SeekEnd {
							offset = s - offset
						}
					} else if s == 0 && offset > 0 {
						offset = 0
					}
					buff.lck.RUnlock()
					buff.lck.Lock()
					defer buff.lck.Unlock()
					if crntbuf := buff.insertedbuffs[offset]; crntbuf != nil {
						crntbuf.Print(a...)
					} else {
						insertbuf := NewBuffer()
						insertbuf.Print(a...)
						buff.insertedbuffs[offset] = insertbuf
					}
				}()
			}
		}
	}
	return
}

// BuffersLen - return len() of internal byte[][] buffer
func (buff *Buffer) BuffersLen() (s int) {
	return len(buff.buffer)
}

// Contains return true if *Buffer contains teststring
func (buff *Buffer) Contains(teststring string) (contains bool) {
	if buff != nil && teststring != "" {
		contains = buff.ContainsBytes([]byte(teststring)...)
	}
	return
}

type checkbytes struct {
	bufcur     *bufferCursor
	testbts    []byte
	testbtsl   int
	testbtsi   int
	FoundMatch func(chkbts *checkbytes, bts ...byte) (fnd bool)
	OnFound    func(offset int64)
	prvb       byte
	dne        bool
}

func newCheckBytes(bufcur *bufferCursor, testbts []byte, OnFound func(int64), FoundMatch func(chkbts *checkbytes, bts ...byte) (fnd bool)) (chkbts *checkbytes) {
	chkbts = &checkbytes{testbts: testbts[:], testbtsl: len(testbts), FoundMatch: FoundMatch, OnFound: OnFound, bufcur: bufcur}
	return
}

func (chkbts *checkbytes) close() {
	if chkbts != nil {
		if chkbts.testbts != nil {
			chkbts.testbts = nil
		}
		if chkbts.bufcur != nil {
			chkbts.bufcur = nil
		}
		if chkbts.FoundMatch != nil {
			chkbts.FoundMatch = nil
		}
		if chkbts.OnFound != nil {
			chkbts.OnFound = nil
		}
	}
}

func (chkbts *checkbytes) foundMatch() (fnd bool) {
	if chkbts != nil && chkbts.bufcur != nil {
		bts, lastbts := chkbts.bufcur.nextBytes()
		if len(bts) > 0 {
			if chkbts.FoundMatch != nil {
				if chkbts.FoundMatch(chkbts, bts...) {
					fnd = true
					if chkbts.OnFound != nil && chkbts.dne {
						if chkbts.bufcur != nil {
							chkbts.OnFound(chkbts.bufcur.lastOffset)
						} else {
							chkbts.OnFound(-1)
						}
					}
				}
			}
		}
		if lastbts {
			fnd = true
		}
	}
	return
}

func foundContains(chkbts *checkbytes, bts ...byte) (fnd bool) {
	for bn, bt := range bts {
		if chkbts.testbtsi > 0 && chkbts.testbts[chkbts.testbtsi-1] == chkbts.prvb && chkbts.testbts[chkbts.testbtsi] != bt {
			chkbts.testbtsi = 0
		}
		if chkbts.testbts[chkbts.testbtsi] == bt {
			chkbts.testbtsi++
			if chkbts.testbtsi == chkbts.testbtsl {
				chkbts.bufcur.lastOffset = chkbts.bufcur.curOffset + int64(bn) - int64(chkbts.testbtsl)
				fnd = true
				chkbts.dne = true
				break
			} else {
				chkbts.prvb = bt
			}
		} else {
			if chkbts.testbtsi > 0 {
				chkbts.testbtsi = 0
			}
			chkbts.prvb = bt
		}
	}
	return
}

func foundContainsReverse(chkbts *checkbytes, bts ...byte) (fnd bool) {
	if btsl := len(bts); btsl > 0 {
		for bi := range bts {
			bt := bts[btsl-(bi+1)]
			if chkbts.testbts[chkbts.testbtsl-(chkbts.testbtsi+1)] == bt {
				chkbts.testbtsi++
				if chkbts.testbtsi == chkbts.testbtsl {
					chkbts.bufcur.lastOffset = chkbts.bufcur.curOffset + int64(btsl-(bi+1))
					fnd = true
					chkbts.dne = true
					break
				} else {
					chkbts.prvb = bt
				}
			} else {
				if chkbts.testbtsi > 0 {
					chkbts.testbtsi = 0
				}
				chkbts.prvb = bt
			}
		}
	}
	return
}

func foundPrefix(chkbts *checkbytes, bts ...byte) (fnd bool) {
	for bn, bt := range bts {
		if chkbts.testbtsi > 0 && chkbts.testbts[chkbts.testbtsi-1] == chkbts.prvb && chkbts.testbts[chkbts.testbtsi] != bt {
			fnd = true
			chkbts.testbtsi = 0
			break
		}
		if chkbts.testbts[chkbts.testbtsi] == bt {
			chkbts.testbtsi++
			if chkbts.testbtsi == chkbts.testbtsl {
				chkbts.bufcur.lastOffset = chkbts.bufcur.curOffset + int64(bn) - int64(chkbts.testbtsl)
				fnd = true
				chkbts.dne = true
				break
			} else {
				chkbts.prvb = bt
			}
		} else {
			fnd = true
			break
		}
	}
	return
}

func foundIndexOf(chkbts *checkbytes, bts ...byte) (fnd bool) {
	for bn, bt := range bts {
		if chkbts.testbtsi > 0 && chkbts.testbts[chkbts.testbtsi-1] == chkbts.prvb && chkbts.testbts[chkbts.testbtsi] != bt {
			chkbts.testbtsi = 0
			chkbts.prvb = 0
		}
		if chkbts.testbts[chkbts.testbtsi] == bt {
			chkbts.testbtsi++
			if chkbts.testbtsi == chkbts.testbtsl {
				chkbts.bufcur.lastOffset = chkbts.bufcur.curOffset + int64(bn) - int64(chkbts.testbtsl)
				fnd = true
				chkbts.dne = true
				break
			} else {
				chkbts.prvb = bt
			}
		} else {
			if chkbts.testbtsi > 0 {
				chkbts.testbtsi = 0
			}
			chkbts.prvb = bt
		}
	}
	return
}

func foundLastIndexOf(chkbts *checkbytes, bts ...byte) (fnd bool) {
	if btsl := len(bts); btsl > 0 {
		for bi := range bts {
			bt := bts[btsl-(bi+1)]
			if chkbts.testbts[chkbts.testbtsl-(chkbts.testbtsi+1)] == bt {
				chkbts.testbtsi++
				if chkbts.testbtsi == chkbts.testbtsl {
					chkbts.bufcur.lastOffset = chkbts.bufcur.curOffset + int64(btsl-(bi+1))
					fnd = true
					chkbts.dne = true
					break
				} else {
					chkbts.prvb = bt
				}
			} else {
				if chkbts.testbtsi > 0 {
					chkbts.testbtsi = 0
				}
				chkbts.prvb = bt
			}
		}
	}
	return
}

func foundSuffix(chkbts *checkbytes, bts ...byte) (fnd bool) {
	if btsl := len(bts); btsl > 0 {
		for btn := range bts {
			bt := bts[btsl-(btn+1)]
			if chkbts.testbts[chkbts.testbtsl-(chkbts.testbtsi+1)] == bt {
				chkbts.testbtsi++
				if chkbts.testbtsi == chkbts.testbtsl {
					chkbts.bufcur.lastOffset = chkbts.bufcur.curOffset + int64(btsl-(btn+1)) - int64(chkbts.testbtsl)
					chkbts.dne = true
					fnd = true
					break
				} else {
					chkbts.prvb = bt
				}
			} else {
				fnd = true
				break
			}
		}
	}
	return
}

// ContainsBytes return true if *Buffer contains testbts...
func (buff *Buffer) ContainsBytes(testbts ...byte) (contains bool) {
	contains = internalContainsBytes(buff, testbts)
	return
}

func internalContainsBytes(buff *Buffer, testbts []byte, offsets ...int64) (contains bool) {
	if testbtsl := len(testbts); testbtsl > 0 && buff != nil && buff.Size() > 0 {
		if len(offsets) == 0 {
			offsets = append(offsets, 0, buff.Size())
		}
		func() {
			var bufcur = newBufferCursor(buff, true, offsets...)
			var chkbts = newCheckBytes(bufcur, testbts, func(offset int64) {
				contains = true
			}, foundContains)
			defer func() {
				chkbts.close()
				chkbts = nil
				bufcur.close()
				bufcur = nil
			}()
			var bufcurback = newBufferCursor(buff, false, offsets...)
			var chkbtsback = newCheckBytes(bufcurback, testbts, func(offset int64) {
				contains = true
			}, foundContainsReverse)

			defer func() {
				chkbtsback.close()
				chkbtsback = nil
				bufcurback.close()
				bufcurback = nil
			}()
			for {
				if chkbts.foundMatch() || chkbtsback.foundMatch() {
					break
				}
			}
		}()
	}
	return
}

// HasPrefix return true if *Buffer has prefix teststring
func (buff *Buffer) HasPrefix(teststring string) (isprefixed bool) {
	if buff != nil && teststring != "" {
		isprefixed = buff.HasPrefixBytes([]byte(teststring)...)
	}
	return
}

type bufferCursor struct {
	buff        *Buffer
	lastBytes   []byte
	noMoreBytes bool
	fromOffset  int64
	toOffset    int64
	buffs       int64
	curOffset   int64
	lastOffset  int64
	//bufi       int
	//bufl       int
	asc bool
}

func (bufcur *bufferCursor) reset(asc bool, offsets ...int64) {
	if bufcur != nil {
		bufcur.buffs = bufcur.buff.Size()
		bufcur.asc = asc
		bufcur.curOffset = -1
		bufcur.lastOffset = -1
		bufcur.fromOffset = -1
		bufcur.toOffset = -1
		if len(offsets) > 0 && len(offsets)%2 == 0 {
			if offsets[0] >= 0 && offsets[1] > 0 && offsets[0] < offsets[1] && offsets[1] <= bufcur.buffs {
				bufcur.fromOffset = offsets[0]
				bufcur.toOffset = offsets[1]
			}
		}
	}
}

func (bufcur *bufferCursor) close() {
	if bufcur != nil {
		if bufcur.buff != nil {
			bufcur.buff = nil
		}
		bufcur = nil
	}
}

func (bufcur *bufferCursor) Read(p []byte) (n int, err error) {
	if bufcur != nil {
		if pl := len(p); pl > 0 {
			lastBtsl := 0
			for n < pl {
				if lastBtsl = len(bufcur.lastBytes); lastBtsl == 0 {
					bufcur.lastBytes, bufcur.noMoreBytes = bufcur.nextBytes()
					if lastBtsl = len(bufcur.lastBytes); lastBtsl == 0 {
						err = io.EOF
						break
					}
				}
				if lastBtsl <= (pl - n) {
					cpn := copy(p[n:], bufcur.lastBytes[:lastBtsl])
					n += cpn
					bufcur.lastBytes = bufcur.lastBytes[lastBtsl-(lastBtsl-cpn):]
				} else if (pl - n) < lastBtsl {
					cpn := copy(p[n:], bufcur.lastBytes[:lastBtsl])
					n += cpn
					bufcur.lastBytes = bufcur.lastBytes[lastBtsl-(lastBtsl-cpn):]
				}
			}
			if n == 0 && err == io.EOF {
				err = io.EOF
			}
		}
	}
	return
}

func (bufcur *bufferCursor) nextBytes() (bts []byte, lastBytes bool) {
	if bufcur != nil {
		if buff := bufcur.buff; buff != nil && bufcur.fromOffset >= 0 && (bufcur.asc || !bufcur.asc) {
			func() {
				buff.lck.RLock()
				defer buff.lck.RUnlock()
				bufs := int64(0)
				if bfl := len(buff.buffer); bfl > 0 {
					bufs = int64(bfl) * int64(len(buff.buffer[0]))
				}
				if bufcur.asc {
					if bufcur.fromOffset < bufcur.toOffset {
						if bufcur.fromOffset < bufs {
							bl := len(buff.buffer[0])
							bfi := int((bufcur.fromOffset + 1) / int64(bl))
							bi := int(bufcur.fromOffset % int64(bl))

							if bufcur.toOffset <= int64(bl) || (bufcur.toOffset <= bufs && bufcur.toOffset > (bufs-int64(bl))) {
								bmi := bl - int(bufcur.toOffset%int64(bl))
								bts = buff.buffer[bfi][bi:bmi]
							} else {
								bts = buff.buffer[bfi][bi:bl]
							}

							bufcur.curOffset = bufcur.fromOffset
							bufcur.fromOffset += int64(len(bts))
						} else if bufcur.fromOffset < bufcur.buffs {
							bts = buff.bytes[int(bufcur.fromOffset-bufs):int(bufcur.toOffset-bufs)]
							bufcur.curOffset = bufcur.fromOffset
							bufcur.fromOffset += int64(len(bts))
						}
					}
					lastBytes = bufcur.fromOffset >= bufcur.toOffset
				} else {
					if bufcur.toOffset > bufcur.fromOffset {
						if bufcur.toOffset <= bufs {
							if bl := len(buff.buffer[0]); bufcur.toOffset < int64(bl) {
								bts = buff.buffer[0][int(bufcur.fromOffset):int(bufcur.toOffset)]
							} else {
								bfi := int((bufcur.toOffset) / int64(bl))
								bmi := int(bufcur.toOffset - (int64(bl) * int64(bfi-1)))
								if bufcur.fromOffset >= (int64(bfi-1) * int64(bl)) {
									bi := int(bufcur.fromOffset % int64(bl))
									bts = buff.buffer[bfi-1][bi:bmi]
								} else {
									bts = buff.buffer[bfi-1][:bmi]
								}
							}
							bufcur.curOffset = bufcur.toOffset - int64(len(bts))
							bufcur.toOffset -= int64(len(bts))
						} else if bufcur.toOffset > bufs {
							if bufcur.fromOffset > bufs {
								bts = buff.bytes[int(bufcur.fromOffset-bufs):int(bufcur.toOffset-bufs)]
							} else {
								bts = buff.bytes[:int(bufcur.toOffset-bufs)]
							}
							bufcur.curOffset = bufcur.toOffset - int64(len(bts))
							bufcur.toOffset -= int64(len(bts))
						}
					}
					lastBytes = bufcur.fromOffset >= bufcur.toOffset
				}
			}()
		} else {
			lastBytes = true
		}
	}
	return
}

func newBufferCursor(buff *Buffer, asc bool, offsets ...int64) (bufcur *bufferCursor) {
	bufcur = &bufferCursor{buff: buff, asc: asc, curOffset: -1, buffs: buff.Size(), fromOffset: -1, toOffset: -1}
	if len(offsets) > 0 && len(offsets)%2 == 0 {
		if offsets[0] >= 0 && offsets[1] > 0 && offsets[0] < offsets[1] && offsets[1] <= bufcur.buffs {
			bufcur.fromOffset = offsets[0]
			bufcur.toOffset = offsets[1]
		}
	}
	return
}

// HasPrefixBytes return true if *Buffer has prefix testbts...
func (buff *Buffer) HasPrefixBytes(testbts ...byte) (isprefixed bool) {
	isprefixed = internalHasPrefixBytes(buff, testbts)
	return
}

func internalHasPrefixBytes(buff *Buffer, testbts []byte, offsets ...int64) (isprefixed bool) {
	if testbtsl := len(testbts); testbtsl > 0 && buff != nil && buff.Size() > 0 {
		if len(offsets) == 0 {
			offsets = append(offsets, 0, buff.Size())
		}
		func() {
			var bufcur = newBufferCursor(buff, true, offsets...)
			var chkbts = newCheckBytes(bufcur, testbts, func(offset int64) {
				isprefixed = true
			}, foundPrefix)
			defer func() {
				chkbts.close()
				bufcur.close()
			}()
			for {
				if chkbts.foundMatch() {
					break
				}
			}
		}()
	}
	return
}

// IndexOf return int64 index of *Buffer prefix teststring else -1 if not found
func (buff *Buffer) IndexOf(teststring string) (index int64) {
	if buff != nil && teststring != "" {
		index = buff.IndexOfBytes([]byte(teststring)...)
	}
	return
}

// LastIndexOf return int64 index of *Buffer prefix teststring else -1 if not found
func (buff *Buffer) LastIndexOf(teststring string) (index int64) {
	if buff != nil && teststring != "" {
		index = buff.LastIndexOfBytes([]byte(teststring)...)
	}
	return
}

// IndexOfBytes return int64 index of *Buffer prefix testbts... else -1 of not found
func (buff *Buffer) IndexOfBytes(testbts ...byte) (index int64) {
	index = internalIndexOfBytes(buff, testbts)
	return
}

// LastIndexOfBytes return int64 index of *Buffer prefix testbts... else -1 of not found
func (buff *Buffer) LastIndexOfBytes(testbts ...byte) (index int64) {
	index = internalLastIndexOfBytes(buff, testbts)
	return
}

func internalIndexOfBytes(buff *Buffer, testbts []byte, offsets ...int64) (index int64) {
	index = -1
	if testbtsl := len(testbts); testbtsl > 0 && buff != nil && buff.Size() > 0 {
		func() {
			if len(offsets) == 0 {
				offsets = append(offsets, 0, buff.Size())
			}
			var bufcur = newBufferCursor(buff, true, offsets...)
			var chkbts = newCheckBytes(bufcur, testbts, func(offset int64) {
				index = offset
			}, foundIndexOf)
			defer func() {
				chkbts.close()
				bufcur.close()
			}()
			for {
				if chkbts.foundMatch() {
					break
				}
			}
		}()
	}
	return
}

func internalLastIndexOfBytes(buff *Buffer, testbts []byte, offsets ...int64) (index int64) {
	index = -1
	if testbtsl := len(testbts); testbtsl > 0 && buff != nil && buff.Size() > 0 {
		func() {
			if len(offsets) == 0 {
				offsets = append(offsets, 0, buff.Size())
			}
			var bufcurback = newBufferCursor(buff, false, offsets...)
			var chkbtsback = newCheckBytes(bufcurback, testbts, func(offset int64) {
				index = offset
			}, foundLastIndexOf)

			defer func() {
				chkbtsback.close()
				chkbtsback = nil
				bufcurback.close()
				bufcurback = nil
			}()
			for {
				if chkbtsback.foundMatch() {
					break
				}
			}
		}()
	}
	return
}

// HasSuffix return true if *Buffer has suffix teststring
func (buff *Buffer) HasSuffix(teststring string) (isprefixed bool) {
	if buff != nil && teststring != "" {
		isprefixed = buff.HasSuffixBytes([]byte(teststring)...)
	}
	return
}

// HasSuffixBytes return true if *Buffer has suffix testbts...
func (buff *Buffer) HasSuffixBytes(testbts ...byte) (issuffixed bool) {
	if testbtsl := len(testbts); testbtsl > 0 && buff != nil && buff.Size() > 0 {
		issuffixed = internalHasSuffixBytes(buff, testbts)
	}
	return
}

func internalHasSuffixBytes(buff *Buffer, testbts []byte, offsets ...int64) (issuffixed bool) {
	if testbtsl := len(testbts); testbtsl > 0 && buff != nil && buff.Size() > 0 {
		if len(offsets) == 0 {
			offsets = append(offsets, 0, buff.Size())
		}
		func() {
			var bufcur = newBufferCursor(buff, false, offsets...)
			var chkbts = *newCheckBytes(bufcur, testbts, func(offset int64) {
				issuffixed = true
			}, foundSuffix)
			defer func() {
				chkbts.close()
				bufcur.close()
			}()
			for {
				if chkbts.foundMatch() {
					break
				}
			}
		}()
	}
	return
}

// Clone - return *Buffer clone
func (buff *Buffer) Clone(clear ...bool) (clnbf *Buffer) {
	clnbf = NewBuffer()
	if buff.Size() > 0 {
		if bfl := len(buff.buffer); bfl > 0 {
			if clnbf.buffer == nil {
				clnbf.buffer = [][]byte{}
			}
			if len(clear) == 1 && clear[0] {
				for bfl > 0 {
					clnbf.buffer = append(clnbf.buffer, buff.buffer[0])
					bfl--
					buff.buffer = buff.buffer[1:]
				}
			} else {
				clnbf.buffer = append(clnbf.buffer, buff.buffer...)
			}
		}
		if buff.bytesi > 0 {
			copy(clnbf.bytes, buff.bytes[:buff.bytesi])
			clnbf.bytesi = buff.bytesi
		}
		if len(clear) == 1 && clear[0] {
			buff.Clear()
		}
	}
	return
}

// Print - same as fmt.Print just on buffer
func (buff *Buffer) Print(a ...interface{}) (err error) {
	err = Fprint(buff, a...)
	return
}

// Println - same as fmt.Println just on buffer
func (buff *Buffer) Println(a ...interface{}) (err error) {
	err = Fprintln(buff, a...)
	return
}

func internalSubWrite(w io.Writer, buff *Buffer, bufcur *bufferCursor, offset ...int64) (err error) {
	if buff != nil && w != nil {
		if len(offset) > 0 && len(offset)%2 == 0 {
			if sl := buff.Size(); sl > 0 {
				var offs int64 = offset[0]
				if offs == -1 {
					offs = 0
				}
				var offe int64 = offset[1]
				if offe == -1 || offe > sl {
					offe = sl
				}
				if offs >= 0 && offe > 0 && offe <= sl && offe > offs {
					if bufcur == nil {
						bufcur = newBufferCursor(buff, true, offs, offe)
						for bufcur.fromOffset < bufcur.toOffset {
							bts, btslst := bufcur.nextBytes()
							pn, pnerr := w.Write(bts)
							if pnerr != nil {
								err = pnerr
								break
							}
							offs += int64(pn)
							if btslst {
								break
							}
						}
					}
				}
			}
		}
	}
	return
}

// SubString - return buffer as string value based on offset ...int64
func (buff *Buffer) SubString(offset ...int64) (s string, err error) {
	if buff != nil {
		if len(offset) == 1 {
			offset = append(offset, buff.Size())
		}
		if len(offset) > 0 && len(offset)%2 == 0 {
			runebts := make([]byte, 4)
			runebtsi := 0
			rns := []rune{}
			_, err = internalBufferWriteToOffSet(buff, nil, offset[0], offset[1], offset[1], func(p []byte) (n int, wtrerr error) {
				for _, b := range p {
					runebts[runebtsi] = b
					runebtsi++
					if r, rs := utf8.DecodeRune(runebts[:runebtsi]); !(r == utf8.RuneError && (rs == 0 || rs == 1)) && rs > 0 {
						rns = append(rns, r)
						if len(rns) == 8192 {
							s += string(rns)
							rns = nil
						}
						runebtsi = 0
					}
					if runebtsi == len(runebts) {
						return 0, io.EOF
					}
				}
				return
			})
			if err == nil {
				if len(rns) > 0 {
					s += string(rns)
					rns = nil
				}
			}
		}
	}
	return
}

// String - return buffer as string value
func (buff *Buffer) String() (s string) {
	s = ""
	if buff != nil {
		err := error(nil)
		if s, err = buff.SubString(0); err != nil {
			s = ""
		}
	}
	return
}

// Size - total size of Buffer
func (buff *Buffer) Size() (s int64) {
	s = 0
	if len(buff.buffer) > 0 {
		s += (int64(len(buff.buffer)) * int64(len(buff.buffer[0])))
	}
	if buff.bytesi > 0 {
		s += int64(buff.bytesi)
	}
	if len(buff.insertedbuffs) > 0 {
		for _, ibuf := range buff.insertedbuffs {
			s += ibuf.Size()
		}
	}
	return s
}

// ReadRunesFrom - refere to io.ReaderFrom
func (buff *Buffer) ReadRunesFrom(r io.Reader) (n int64, err error) {
	if r != nil {
		var rnsr io.RuneReader = nil
		if rnsr, _ = r.(io.RuneReader); rnsr == nil {
			rnsr = bufio.NewReader(r)
		}
		var p = make([]rune, 4096)
		var ppi = 0
		for {
			pr, pn, pnerr := rnsr.ReadRune()
			if pn > 0 {
				n += int64(pn)
				p[ppi] = pr
				ppi++
				if ppi == len(p) {
					var pi = 0
					for pi < ppi {
						wn, wnerr := buff.WriteRunes(p[pi : pi+(ppi-pi)]...)
						if wn > 0 {
							pi += wn
						}
						if wnerr != nil {
							pnerr = wnerr
							break
						}
						if wn == 0 {
							break
						}
					}
				}
			}
			if pnerr != nil {
				err = pnerr
				break
			} else {
				if pn == 0 {
					err = io.EOF
					break
				}
			}
		}
		if ppi > 0 {
			var pi = 0
			for pi < ppi {
				wn, wnerr := buff.WriteRunes(p[pi : pi+(ppi-pi)]...)
				if wn > 0 {
					pi += wn
				}
				if wnerr != nil {
					err = wnerr
					break
				}
				if wn == 0 {
					break
				}
			}
		}
	}
	return
}

// ReadFrom - fere io.ReaderFrom
func (buff *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	if r != nil {
		var p = make([]byte, 4096)
		for {
			pn, pnerr := r.Read(p)
			if pn > 0 {
				n += int64(pn)
				var pi = 0
				for pi < pn {
					wn, wnerr := buff.Write(p[pi : pi+(pn-pi)])
					if wn > 0 {
						pi += wn
					}
					if wnerr != nil {
						pnerr = wnerr
						break
					}
					if wn == 0 {
						break
					}
				}
			}
			if pnerr != nil {
				err = pnerr
				break
			} else {
				if pn == 0 {
					err = io.EOF
					break
				}
			}
		}
	}
	return
}

// WriteRune - Write singe rune
func (buff *Buffer) WriteRune(r rune) (err error) {
	if bs := RunesToUTF8(r); len(bs) > 0 {
		_, err = buff.Write(bs)
	}
	return
}

// WriteRunes - Write runes
func (buff *Buffer) WriteRunes(p ...rune) (n int, err error) {
	if pl := len(p); pl > 0 {
		if bs := RunesToUTF8(p[:pl]...); len(bs) > 0 {
			_, err = buff.Write(bs)
		}
		n = pl
	}
	return
}

func internalBufferWriteToOffSet(buff *Buffer, w io.Writer, stroffset, endoffset, maxoffset int64, wrtbtsfunc func([]byte) (int, error)) (n int64, err error) {
	if buff != nil && stroffset >= 0 && stroffset < endoffset && maxoffset > 0 && maxoffset <= buff.Size() {
		if wbuf, _ := w.(*Buffer); wbuf != nil {
			var wrtbst = func(b []byte) {
				wn, _ := wbuf.Write(b)
				n += int64(wn)
			}

			for btsn, bts := range buff.buffer {
				btnbufoffset, btnbuffoffsetlen := int64(len(buff.buffer[0])*btsn), int64(len(buff.buffer[0]))*int64(btsn+1)
				if btnbufoffset < endoffset {
					if stroffset >= btnbufoffset {
						if stroffset < btnbufoffset+btnbuffoffsetlen {
							btsn = int(stroffset - btnbufoffset)
							if endoffset <= btnbuffoffsetlen {
								btsne := int(endoffset - btnbufoffset)
								wrtbst(bts[btsn:btsne])
								return
							} else {
								wrtbst(bts[btsn:])
							}
						} else {
							return
						}
					} else if btnbufoffset > stroffset {
						if endoffset <= btnbuffoffsetlen {
							btsne := int(endoffset - btnbufoffset)
							wrtbst(bts[:btsne])
							return
						} else if endoffset > btnbuffoffsetlen {
							wrtbst(bts)
						} else {
							return
						}
					}
				} else {
					return
				}
			}
			if buff.bytesi > 0 {
				bts := buff.bytes[:buff.bytesi]
				btnbufoffset := int64(0)
				if len(buff.buffer) > 0 {
					btnbufoffset = int64(len(buff.buffer) * len(buff.buffer[0]))
				}
				btnbuffoffsetlen := btnbufoffset + int64(buff.bytesi)
				if btnbufoffset < endoffset {
					if stroffset >= btnbufoffset {
						if stroffset < btnbufoffset+btnbuffoffsetlen {
							btsn := int(stroffset - btnbufoffset)
							if endoffset <= btnbuffoffsetlen {
								btsne := int(endoffset - btnbufoffset)
								wrtbst(bts[btsn:btsne])
								return
							} else {
								wrtbst(bts[btsn:])
							}
						} else {
							return
						}
					} else if btnbufoffset > stroffset {
						if endoffset <= btnbuffoffsetlen {
							btsne := int(endoffset - btnbufoffset)
							wrtbst(bts[:btsne])
							return
						} else if endoffset > btnbuffoffsetlen {
							wrtbst(bts)
						} else {
							return
						}
					}
				} else {
					return
				}
			}
			return
		} else {
			var bufcur = newBufferCursor(buff, true, stroffset, endoffset)
			defer bufcur.close()
			if wrtbtsfunc != nil {
				for {
					bts, lstbytes := bufcur.nextBytes()
					if wn, werr := wrtbtsfunc(bts); werr != nil {
						err = werr
						return
					} else if wn > 0 {
						n += int64(wn)
					}
					if lstbytes {
						break
					}
				}
			} else {
				if err = Fprint(w, bufcur); err == nil {
					n = buff.Size()
				}
			}
		}
	}
	return
}

func (buff *Buffer) WriteSubOffsetTo(w io.Writer, offsets ...int64) (n int64, err error) {
	if bufs := buff.Size(); bufs > 0 && len(offsets) > 0 {
		stroffset := offsets[0]

		if len(offsets) == 1 {
			offsets = append(offsets, buff.Size())
		}
		n, err = internalBufferWriteToOffSet(buff, w, stroffset, offsets[1], bufs, nil)
	}
	return
}

func (buff *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	if bufs := buff.Size(); bufs > 0 {
		n, err = internalBufferWriteToOffSet(buff, w, 0, bufs, bufs, nil)
	}
	return
}

// Write - refer io.Writer
func (buff *Buffer) Write(p []byte) (n int, err error) {
	if pl := len(p); pl > 0 {
		func() {
			buff.lck.Lock()
			defer buff.lck.Unlock()
			for n < pl {
				if tl := (len(buff.bytes) - buff.bytesi); (pl - n) >= tl {
					if cl := copy(buff.bytes[buff.bytesi:buff.bytesi+tl], p[n:n+tl]); cl > 0 {
						n += cl
						buff.bytesi += cl
						if buff.MaxLenToWrite > 0 {
							if buff.maxwrttnl < 0 {
								buff.maxwrttnl = int64(cl)
							} else {
								buff.maxwrttnl += int64(cl)
							}
						}
					}
				} else if tl := (pl - n); tl < (len(buff.bytes) - buff.bytesi) {
					if cl := copy(buff.bytes[buff.bytesi:buff.bytesi+tl], p[n:n+tl]); cl > 0 {
						n += cl
						buff.bytesi += cl
						if buff.MaxLenToWrite > 0 {
							if buff.maxwrttnl < 0 {
								buff.maxwrttnl = int64(cl)
							} else {
								buff.maxwrttnl += int64(cl)
							}
						}
					}
				}
				if buff.bytesi == len(buff.bytes) {
					if buff.buffer == nil {
						buff.buffer = [][]byte{}
					}
					var bts = make([]byte, buff.bytesi)
					copy(bts, buff.bytes[:buff.bytesi])
					buff.buffer = append(buff.buffer, bts)
					buff.bytesi = 0
				}
			}
		}()
		if buff.MaxLenToWrite > 0 && buff.maxwrttnl >= buff.MaxLenToWrite {
			if buff.OnMaxWritten != nil {
				if buff.OnMaxWritten(buff.maxwrttnl) {
					buff.maxwrttnl = -1
				} else {
					buff.MaxLenToWrite = -1
					buff.maxwrttnl = -1
				}
			} else {
				buff.MaxLenToWrite = -1
				buff.maxwrttnl = -1
			}
		}
	}
	return
}

// Reader -
func (buff *Buffer) Reader(args ...interface{}) (bufr *BuffReader) {
	var offset []int64 = nil
	var disposeBuffer bool = false
	for _, d := range args {
		if d != nil {
			if bd, _ := d.(bool); bd {
				if !disposeBuffer {
					disposeBuffer = true
				}
			} else if int64d, _ := d.(int64); int64d >= 0 && len(offset) < 2 {
				offset = append(offset, int64d)
			} else if int64offsetsd, _ := d.([]int64); len(int64offsetsd) > 0 && len(offset) == 0 {
				offset = append(offset, int64offsetsd...)
			}
		}
	}
	if buff != nil {
		if len(offset) == 0 && buff.Size() > 0 {
			offset = append(offset, 0, buff.Size())
		} else if len(offset) == 1 && buff.Size() > 0 {
			offset = append(offset, buff.Size())
		}
		bufr = &BuffReader{buffer: buff /* roffset: -1,*/, bufcur: newBufferCursor(buff, true, offset...), MaxRead: -1, Options: map[string]string{}}
		bufr.DisposeBuffer = disposeBuffer
	}
	return
}

// Close - refer io.Closer
func (buff *Buffer) Close() (err error) {
	if buff != nil {
		if buff.lck != nil {
			if buff.OnClose != nil {
				buff.OnClose(buff)
				buff.OnClose = nil
			}
			buff.Clear()
			buff.lck = nil
		}
		buff = nil
	}
	return
}

type readerdisposed func()

// Clear - Buffer
func (buff *Buffer) Clear() (err error) {
	if buff != nil {
		var rdrdisposed = []readerdisposed{}
		if buff.lck != nil {
			func() {
				buff.lck.Lock()
				defer buff.lck.Unlock()

				if buff.bufrs != nil {
					if len(buff.bufrs) > 0 {
						var bufrs = make([]*BuffReader, len(buff.bufrs))
						var bufrsi = 0
						for bufrsk := range buff.bufrs {
							buff.bufrs[bufrsk] = nil
							bufrs[bufrsi] = bufrsk
							if bufrsk.Disposed != nil {
								rdrdisposed = append(rdrdisposed, bufrsk.Disposed)
								bufrsk.Disposed = nil
							}
							bufrsk.Close()
							bufrsi++
						}
						for bufrskn := range bufrs {
							delete(buff.bufrs, bufrs[bufrskn])
						}
						bufrs = nil
					}
					buff.bufrs = nil
				}
				if buff.buffer != nil {
					for len(buff.buffer) > 0 {
						buff.buffer[0] = nil
						buff.buffer = buff.buffer[1:]
					}
					buff.buffer = nil
				}
				if buff.bytesi > 0 {
					buff.bytesi = 0
				}
				if buff.MaxLenToWrite > 0 {
					if buff.OnMaxWritten == nil {
						buff.MaxLenToWrite = -1
						buff.maxwrttnl = -1
					}
				}
				if len(buff.insertedbuffs) > 0 {
					for off, offbuf := range buff.insertedbuffs {
						offbuf.Close()
						delete(buff.insertedbuffs, off)
					}
				}
			}()
		}
		if len(rdrdisposed) > 0 {
			for _, rdrdsp := range rdrdisposed {
				rdrdsp()
			}
			rdrdisposed = nil
		}
	}
	return
}

func (buff *Buffer) Array(args ...interface{}) (arr []interface{}, err error) {
	if buff != nil {
		if buffr := buff.Reader(args...); buffr != nil {
			defer buffr.Close()
			arr, err = buffr.Array()
		}
	}
	return
}

func (buff *Buffer) Map(args ...interface{}) (mp map[string]interface{}, err error) {
	if buff != nil {
		if buffr := buff.Reader(args...); buffr != nil {
			defer buffr.Close()
			mp, err = buffr.Map()
		}
	}
	return
}

// BuffReader -
type BuffReader struct {
	buffer  *Buffer
	rnr     *bufio.Reader
	MaxRead int64
	//roffset  int64
	bufcur        *bufferCursor
	rbytes        []byte
	rbytesi       int
	Disposed      readerdisposed
	Options       map[string]string
	DisposeBuffer bool
	DisposeReader bool
}

// DisposeEOFReader - indicate when reader reach EOF then bufr.Close()
func (bufr *BuffReader) DisposeEOFReader() *BuffReader {
	if bufr != nil {
		bufr.DisposeReader = true
	}
	return bufr
}

// SetMaxRead - set max read implementation for Reader interface compliance
func (bufr *BuffReader) SetMaxRead(maxlen int64) (err error) {
	if bufr != nil {
		if maxlen < 0 {
			maxlen = -1
		}
		bufr.MaxRead = maxlen
	}
	return
}

func (bufr *BuffReader) CanRead() (canread bool) {
	if bufr != nil {
		//if bufr.MaxRead > 0 {
		//	canread = true
		//} else {
		if bufr.rnr != nil {
			if canread = bufr.rnr.Buffered() > 0; !canread {
				if _, peekerr := bufr.rnr.Peek(2); peekerr == nil {
					canread = bufr.rnr.Buffered() > 0
				}
			}
		} else if bufr.bufcur != nil {
			canread = bufr.bufcur.fromOffset < bufr.bufcur.toOffset
		}
		//}
	}
	return
}

func (bufr *BuffReader) WriteToFunc(funcw func([]byte) (int, error)) (n int64, err error) {
	if bufr != nil && funcw != nil {
		n, err = WriteToFunc(bufr, funcw)
	}
	return
}

// WriteTo - helper for io.Copy
func (bufr *BuffReader) WriteTo(w io.Writer) (n int64, err error) {
	if w != nil && bufr != nil {
		var r = io.Reader(bufr)
		if bufr.rnr != nil {
			r = bufr.rnr
		}
		var p = make([]byte, 4096)
		for {
			pn, pnerr := r.Read(p)
			if pn > 0 {
				n += int64(pn)
				var pi = 0
				for pi < pn {
					wn, wnerr := w.Write(p[pi : pi+(pn-pi)])
					if wn > 0 {
						pi += wn
					}
					if wnerr != nil {
						pnerr = wnerr
						break
					}
					if wn == 0 {
						break
					}
				}
			}
			if pnerr == nil {
				if pn == 0 {
					pnerr = io.EOF
				}
			}
			if pnerr != nil {
				err = pnerr
				break
			}
		}

	}
	return
}

// Close - refer io.Closer
func (bufr *BuffReader) Close() (err error) {
	if bufr != nil {
		if bufr.buffer != nil {
			func() {
				if _, ok := bufr.buffer.bufrs[bufr]; ok {
					bufr.buffer.bufrs[bufr] = nil
					delete(bufr.buffer.bufrs, bufr)
				}
			}()
			if bufr.DisposeBuffer {
				bufr.DisposeBuffer = false
				bufr.buffer.Close()
			}
			bufr.buffer = nil
		}
		if bufr.DisposeReader {
			bufr.DisposeReader = false
		}
		if bufr.rnr != nil {
			bufr.rnr = nil
		}
		if bufr.rbytes != nil {
			bufr.rbytes = nil
		}
		if bufr.Options != nil {
			for bfrk := range bufr.Options {
				delete(bufr.Options, bfrk)
			}
			bufr.Options = nil
		}
		if bufr.bufcur != nil {
			bufr.bufcur.close()
			bufr.bufcur = nil
		}
		if bufr.Disposed != nil {
			bufr.Disposed()
			bufr.Disposed = nil
		}
	}
	return
}

// RuneAt - rune at offset int64
func (bufr *BuffReader) RuneAt(offset int64) (rn rune) {
	rn = -1
	if s := bufr.SubString(offset, offset); s != "" {
		rn = rune(s[0])
	}
	return
}

// LastIndex - Last index of s string - n int64
func (bufr *BuffReader) LastIndex(s string, offset ...int64) int64 {
	if bufr == nil || s == "" {
		return -1
	}
	if len(offset) == 2 {
		return bufr.LastByteIndexWithinOffsets(offset[0], offset[1], []byte(s)...)
	} else if len(offset) == 1 {
		return bufr.LastByteIndexWithinOffsets(-1, offset[0], []byte(s)...)
	}
	return bufr.LastByteIndexWithinOffsets(-1, -1, []byte(s)...)
}

// LastByteIndexWithinOffsets - Last index of bs byte... - n int64 within startoffset and endoffset
func (bufr *BuffReader) LastByteIndexWithinOffsets(startoffset, endoffset int64, bs ...byte) (index int64) {
	index = -1
	if bufr != nil && bufr.buffer != nil && len(bs) > 0 {
		if ls := bufr.buffer.Size(); ls > 0 {
			for i, j := 0, len(bs)-1; i < j; i, j = i+1, j-1 {
				bs[i], bs[j] = bs[j], bs[i]
			}
			prvb := byte(0)
			bsi := 0
			toffset := int64(0)
			if bufr.buffer.bytesi > 0 {
				bti := bufr.buffer.bytesi - 1
				for bti > -1 {
					toffset++
					bt := bufr.buffer.bytes[bti]
					bti--
					if bsi > 0 && bs[bsi-1] == prvb && bs[bsi] != bt {
						bsi = 0
						prvb = byte(0)
					}
					if bs[bsi] == bt {
						bsi++
						if bsi == len(bs) {
							toffset += int64(len(bs))
							index = bufr.buffer.Size() - toffset
							break
						} else {
							prvb = bt
						}
					} else {
						if bsi > 0 {
							bsi = 0
						}
					}
				}
			}
			if index == -1 && len(bufr.buffer.buffer) > 0 {
				bfi := len(bufr.buffer.buffer) - 1
				for bfi > -1 {
					toffset++
					bf := bufr.buffer.buffer[bfi]
					bti := len(bf) - 1
					for bti > -1 {
						bt := bufr.buffer.bytes[bti]
						bti--
						if bsi > 0 && bs[bsi-1] == prvb && bs[bsi] != bt {
							bsi = 0
							prvb = byte(0)
						}
						if bs[bsi] == bt {
							bsi++
							if bsi == len(bs) {
								toffset += int64(len(bs))
								index = bufr.buffer.Size() - toffset
								break
							} else {
								prvb = bt
							}
						} else {
							if bsi > 0 {
								bsi = 0
							}
						}
					}
					if index > -1 {
						break
					}
					bfi--
				}
			}
		}
	}
	return
}

// Index - Index of s string - n int64
func (bufr *BuffReader) Index(s string) int64 {
	if bufr == nil || s == "" {
		return -1
	}
	return bufr.ByteIndex([]byte(s)...)
}

// ByteIndex - Index of bs ...byte - n int64
func (bufr *BuffReader) ByteIndex(bs ...byte) (index int64) {
	index = -1
	if bufr != nil && bufr.buffer != nil && len(bs) > 0 {
		prvb := byte(0)
		bsi := 0
		toffset := int64(-1)
		if len(bufr.buffer.buffer) > 0 {
			for bfn := range bufr.buffer.buffer {
				for btn := range bufr.buffer.buffer[bfn] {
					bt := bufr.buffer.buffer[bfn][btn]
					toffset++
					if bsi > 0 && bs[bsi-1] == prvb && bs[bsi] != bt {
						bsi = 0
						prvb = byte(0)
					}
					if bs[bsi] == bt {
						bsi++
						if bsi == len(bs) {
							index = toffset - int64(len(bs))
							break
						} else {
							prvb = bt
						}
					} else {
						if bsi > 0 {
							bsi = 0
						}
					}
				}
				if index > -1 {
					break
				}
			}
		}
		if index == -1 && bufr.buffer.bytesi > 0 {
			for _, bt := range bufr.buffer.bytes[:bufr.buffer.bytesi] {
				toffset++
				if bsi > 0 && bs[bsi-1] == prvb && bs[bsi] != bt {
					bsi = 0
					prvb = byte(0)
				}
				if bs[bsi] == bt {
					bsi++
					if bsi == len(bs) {
						index = toffset - int64(len(bs))
						break
					} else {
						prvb = bt
					}
				} else {
					if bsi > 0 {
						bsi = 0
					}
				}
			}
		}
	}
	return
}

// Read - refer io.Reader
func (bufr *BuffReader) Reset() {
	if bufr != nil {
		bufr.MaxRead = -1
		bufr.rbytes = nil
		bufr.rbytesi = 0
		if bufr.bufcur == nil {
			bufr.bufcur.reset(true, 0, bufr.buffer.Size())
		}
	}
}

func nextReaderBytes(bufr *BuffReader) (bts []byte, lastBytes bool) {
	if bufr != nil {
		if bufcur := bufr.bufcur; bufcur != nil {
			bts, lastBytes = bufcur.nextBytes()
		} else {
			lastBytes = true
		}
	} else {
		lastBytes = true
	}
	return
}

// Read - refer io.Reader
func (bufr *BuffReader) Read(p []byte) (n int, err error) {
	if pl := len(p); pl > 0 {
		rl := 0
		if bufr != nil {
			if bufr.MaxRead > 0 || bufr.MaxRead == -1 {
				for n < pl && (bufr.MaxRead > 0 || bufr.MaxRead == -1) {
					if len(bufr.rbytes) == 0 || (len(bufr.rbytes) > 0 && len(bufr.rbytes) == bufr.rbytesi) {
						if bufr.bufcur.curOffset == -1 {
							if bts, btslst := nextReaderBytes(bufr); len(bts) > 0 {
								bufr.rbytes = bts[:]
								bufr.rbytesi = 0
							} else if btslst {
								break
							}
						} else {
							if bts, btslst := nextReaderBytes(bufr); len(bts) > 0 {
								bufr.rbytes = bts[:]
								bufr.rbytesi = 0
							} else if btslst {
								break
							}
						}
					}

					for (bufr.MaxRead > 0 || bufr.MaxRead == -1) && (pl > n) && (len(bufr.rbytes) > bufr.rbytesi) {
						rbtsl := len(bufr.rbytes)
						if bufr.MaxRead > 0 {
							if ln := int64(rbtsl - bufr.rbytesi); ln > bufr.MaxRead {
								rl = int(bufr.MaxRead)
							} else {
								rl = int(ln)
							}
							if (rl + bufr.rbytesi) < rbtsl {
								rbtsl = (rl + bufr.rbytesi)
							}
						}
						var cl = 0
						for n < pl && bufr.rbytesi < rbtsl {
							if cl, n, bufr.rbytesi = CopyBytes(p[:pl], n, bufr.rbytes[:rbtsl], bufr.rbytesi); cl > 0 {
								if bufr.MaxRead > 0 {
									bufr.MaxRead -= int64(cl)
									if bufr.MaxRead < 0 {
										bufr.MaxRead = 0
									}
								}
							}
						}
					}
				}
			}
		}
		if n == 0 && err == nil {
			err = io.EOF
			if dspbuf, dsprdr := bufr.DisposeBuffer, bufr.DisposeReader; dspbuf || dsprdr {
				bufr.Close()
			}
		}
	}
	return
}

// SubString - return buffer as string value based on offset ...int64
func (bufr *BuffReader) SubString(offset ...int64) (s string) {
	if bufr != nil && bufr.buffer != nil {
		if len(offset)%2 == 1 && offset[len(offset)-1] >= 0 {
			if bufr.MaxRead > 0 {
				offset = append(offset, offset[len(offset)-1]+bufr.MaxRead)
			} else {
				offset = append(offset, offset[len(offset)-1]+bufr.buffer.Size())
			}
		}
		if len(offset) > 0 && len(offset)%2 == 0 {
			if sl := bufr.buffer.Size(); sl > 0 {
				var offs int64 = offset[0]
				if offs == -1 {
					offs = 0
				}
				var offe int64 = offset[1]
				if offe == -1 {
					offe = sl - 1
				}
				rns := make([]rune, 1024)
				rnsi := 0
				busy := true
				for len(offset) > 0 && busy {
					if offs <= sl && offe <= sl {
						bufr.Seek(offs, 0)
						for {
							r, rs, rerr := bufr.ReadRune()
							if rs > 0 {
								rns[rnsi] = r
								rnsi++
								if rnsi == len(rns) {
									rnsi = 0
									s += string(rns[:])
								}
							}
							if rerr != nil {
								busy = false
								break
							}
							offs++
							if offs >= offe {
								busy = false
								break
							}
						}
						if busy {
							offset = offset[2:]
						}
					} else {
						break
					}
				}
				if rnsi > 0 {
					s += string(rns[:rnsi])
				}
			}
		}
	}
	return
}

// ReadRune - refer io.RuneReader
func (bufr *BuffReader) ReadRune() (r rune, size int, err error) {
	if bufr != nil && bufr.bufcur != nil {
		if bufr.rnr == nil {
			bufr.rnr = bufio.NewReader(bufr)
		}
		r, size, err = bufr.rnr.ReadRune()
	} else {
		err = io.EOF
	}
	return
}

func (bufr *BuffReader) Readln() (ln string, err error) {
	ln, err = ReadLine(bufr)
	return
}

func (bufr *BuffReader) Readlines() (lines []string, err error) {
	for {
		ln, lnerr := bufr.Readln()
		if lnerr == nil {
			if ln != "" {
				if lines == nil {
					lines = []string{}
				}
				lines = append(lines, ln)
			}
		} else {
			break
		}
	}
	return
}

func (bufr *BuffReader) ReadAll() (string, error) {
	return ReaderToString(bufr)
}

// Seek - refer to io.Seeker
func (bufr *BuffReader) Seek(offset int64, whence int) (n int64, err error) {
	if bufr != nil && bufr.buffer != nil {
		var adjusted = false
		if bufs := bufr.buffer.Size(); bufs > 0 {
			func() {
				bufr.buffer.lck.RLock()
				defer bufr.buffer.lck.RUnlock()
				var adjustOffsetRead = func() {
					bufr.bufcur.reset(true, n, bufs)
					adjusted = true
				}
				var rajust = int64(0)
				if bufr.rbytesi > 0 {
					rajust += int64(bufr.rbytesi)
				}
				if whence == io.SeekStart {
					if offset >= 0 && offset < bufs {
						n = offset
						adjustOffsetRead()
					}
				} else if whence == io.SeekCurrent {
					if (bufr.bufcur.curOffset-rajust) >= -1 && ((bufr.bufcur.curOffset-rajust)+offset) < bufs {
						if bufr.bufcur.curOffset == -1 {
							n = bufr.bufcur.curOffset + 1 + offset
						} else {
							n = (bufr.bufcur.curOffset - rajust) + offset
						}
						adjustOffsetRead()
					}
				} else if whence == io.SeekEnd {
					if (bufs-offset) >= 0 && (bufs-offset) <= bufs {
						if (bufs - offset) < bufs {
							n = (bufs - offset)
						} else {
							n = (bufs - offset)
						}
						adjustOffsetRead()
					}
				}
			}()
		}
		if !adjusted {
			n = -1
		} else {
			if bufr.rnr != nil {
				bufr.rnr.Reset(bufr)
			}
		}
	} else {
		n = -1
	}
	return
}

func (bufr *BuffReader) Array() (arr []interface{}, err error) {
	if bufr != nil {
		if arr == nil {
			arr = []interface{}{}
		}
		err = json.NewDecoder(bufr).Decode(&arr)
	}
	return
}

func (bufr *BuffReader) Map() (mp map[string]interface{}, err error) {
	if bufr != nil {
		if mp == nil {
			mp = map[string]interface{}{}
		}
		err = json.NewDecoder(bufr).Decode(&mp)
	}
	return
}
