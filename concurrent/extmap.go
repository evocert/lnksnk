package concurrent

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/evocert/lnksnk/reflection"
)

type Map struct {
	elmmp *sync.Map
	cnt   atomic.Int64
}

func NewMap() (enmmp *Map) {
	enmmp = &Map{elmmp: &sync.Map{}}
	return
}

func (enmmp *Map) Count() (cnt int) {
	if enmmp != nil {
		cnt = int(enmmp.cnt.Load())
	}
	return
}

func (enmmp *Map) Del(key ...interface{}) {
	if keysl := len(key); enmmp != nil && keysl > 0 {
		if elmmp := enmmp.elmmp; elmmp != nil {
			var donedel func([]interface{}, []interface{}) = nil
			if donedel, _ = key[keysl-1].(func([]interface{}, []interface{})); donedel != nil {
				keysl--
				key = key[:keysl]
			}
			delkeys := make([]interface{}, keysl)
			delvalues := make([]interface{}, keysl)
			delkeysi := 0
			elmmp.Range(func(ke, value any) bool {
				for kn, k := range key {
					if k == ke {
						if delval, keyexisted := elmmp.LoadAndDelete(k); keyexisted {
							enmmp.cnt.Add(-1)
							delkeys[delkeysi] = k
							delvalues[delkeysi] = delval
							delkeysi++
						}
						key = append(key[:kn], key[kn+1:]...)
						keysl--
						break
					}
				}
				return keysl > 0
			})

			if delkeysi > 0 && donedel != nil {
				donedel(delkeys[:delkeysi], delvalues[:delkeysi])
			}
		}
	}
}

func (enmmp *Map) Find(k ...interface{}) (value interface{}, found bool) {
	if len(k) == 0 {
		if ks, _ := k[0].(string); ks != "" && strings.Contains(ks, ",") {
			ksarr := strings.Split(ks, ",")
			k = make([]interface{}, len(ksarr))
			for kn, kv := range ksarr {
				k[kn] = kv
			}
		}
	}
	found = findvalue(enmmp, nil, func(val interface{}) {
		value = val
	}, k...)
	return
}

func findvalue(enmmp *Map, slce *Slice, onfound func(value interface{}), k ...interface{}) (found bool) {
	if enmmp == nil && slce == nil {
		return
	}
	if kl := len(k); onfound != nil && kl > 0 {
		var nextelmp = func(enmp *Map, slc *Slice) *sync.Map {
			if enmp != nil {
				return enmmp.elmmp
			} else if slc != nil {
				return slc.elmmp
			}
			return nil
		}
		var value interface{} = nil
		for kn, key := range k {
			if elmp := nextelmp(enmmp, slce); elmp != nil {
				if value, found = elmp.Load(key); found {
					if kl-1 == kn {
						onfound(value)
						return
					} else if enmmp, found = value.(*Map); found && enmmp != nil {
						slce = nil
						continue
					} else if slce, found = value.(*Slice); found && slce != nil {
						enmmp = nil
						continue
					}
					return false
				}
				return false
			}
			return false
		}
	}
	return
}

func (enmmp *Map) Invoke(key any, method string, a ...interface{}) (result []interface{}) {
	if method != "" {
		if val, valfnd := enmmp.Get(key); valfnd && val != nil {
			_, result = reflection.ReflectCallMethod(val, method, a...)
		}
	}
	return
}

func (enmmp *Map) Field(key any, field string, a ...interface{}) (result []interface{}) {
	if field != "" {
		if val, valfnd := enmmp.Get(key); valfnd && val != nil {
			_, result = reflection.ReflectCallField(val, field, a...)
		}
	}
	return
}

func (enmmp *Map) Get(key interface{}) (value interface{}, loaded bool) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil {
			value, loaded = elmmp.Load(key)
		}
	}
	return
}

func (enmmp *Map) Range(ietrfunc func(key, value any) bool) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil && ietrfunc != nil {
			elmmp.Range(ietrfunc)
		}
	}
}

func (enmmp *Map) ForEach(eachitem func(interface{}, interface{}, bool, bool) bool) {
	if enmmp != nil && eachitem != nil {
		if mp := enmmp; mp != nil {
			first := true
			cnt := mp.Count()
			kn := 0
			mp.Range(func(k interface{}, v interface{}) (stop bool) {
				stop = !eachitem(v, k, first, cnt-1 == kn)
				if first {
					first = false
				}
				kn++
				return stop || cnt-1 == kn
			})
		}
	}
}

func (enmmp *Map) Set(key interface{}, value interface{}) {
	if enmmp != nil {
		if elmmp := enmmp.elmmp; elmmp != nil {
			var oldval, loaded = elmmp.Load(key)
			if !loaded {
				enmmp.cnt.Add(1)
			}
			if oldval != value {
				elmmp.Store(key, value)
			} else {
				loaded = false
			}
			if loaded {
				if oldval != nil {
					oldval = nil
				}
			}
		}
	}
}

func (enmmp *Map) Dispose() {
	if enmmp != nil {
		if elmp := enmmp.elmmp; elmp != nil {
			enmmp.elmmp = nil

			delkvs := map[interface{}]interface{}{}
			elmp.Range(func(ke, value any) bool {
				delkvs[ke] = value
				elmp.Delete(ke)
				return false
			})
			if len(delkvs) > 0 {
				for k, v := range delkvs {
					delete(delkvs, k)
					if v != nil {
						v = nil
					}
				}
			}
			elmp = nil
		}
	}
}
