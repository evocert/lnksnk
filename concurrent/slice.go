package concurrent

import (
	"sort"
	"strings"

	"github.com/lnksnk/lnksnk/reflection"
)

type Slice struct {
	*Map
}

func NewSlize() (slce *Slice) {
	slce = &Slice{Map: NewMap()}
	return
}

func (slce *Slice) Append(a ...interface{}) {
	if al := len(a); slce != nil && al > 0 {
		if mp := slce.Map; mp != nil {
			adjustby := mp.Count()
			for an := range a {
				mp.Set(adjustby+an, constructValue(a[an]))
			}
		}
	}
}

func (slce *Slice) Del(index ...int) {
	if ixl := len(index); ixl > 0 {
		ix := 0
		if cnt := slce.Map.Count(); cnt >= ixl {
			for ix < ixl {
				if index[ix] < 0 || index[ix] >= cnt {
					index = append(index[:ix], index[ix+1:]...)
					ixl--
					continue
				}
				ix++
			}
			if ixl > 0 {
				sort.Slice(index, func(i, j int) bool { return index[i] < index[j] })
				a := make([]interface{}, ixl+1)
				for ixn, ix := range index {
					a[ixn] = ix
				}

				a[ixl] = func(keys []int, vals []interface{}) {
					cntdiff := cnt - slce.Map.Count()
					cnt = cnt - cntdiff
					keysl := len(keys)
					ki := 0
					for ; ki < keys[0]; ki++ {
					}
					adjstk := 0
					for keyi := 0; keyi < keysl; keyi++ {
						if dk := keys[keyi]; ki+adjstk == dk {
							if keyi < keysl-1 {
								adjstk++
								for ki+adjstk-1 < keys[keyi+1]-1 {
									if nv, nvok := slce.Map.Get(ki + adjstk); nvok {
										slce.Map.Set(ki, nv)
										ki++
									}
								}
							} else {
								adjstk++
								if nv, nvok := slce.Map.Get(ki + adjstk); nvok {
									slce.Map.Set(ki, nv)
									ki++
								}
								if adjstk == cntdiff {
									for ; ki < cnt+cntdiff; ki++ {
										if ki+cntdiff-1 < cnt+cntdiff-1 {
											if nv, nvok := slce.Map.Get(ki + cntdiff); nvok {
												slce.Map.Set(ki, nv)
											}
										} else {
											slce.Map.Del(ki)
										}
									}
								}
							}
						} else {
							break
						}
					}
				}
				slce.Map.Del(a...)
			}
		}
	}
}

func (slce *Slice) Get(index int) (value interface{}) {
	if slce != nil && index >= 0 {
		if mp := slce.Map; mp != nil {
			if val, valok := mp.Get(index); valok {
				value = val
			}
		}
	}
	return
}

func (slce *Slice) Range(index ...int) (vals []interface{}) {
	if il := len(index); slce != nil && il > 0 {
		if mp := slce.Map; mp != nil {
			sort.Slice(index, func(i, j int) bool { return index[i] < index[j] })
			mp.Range(func(k interface{}, v interface{}) (stop bool) {
				if index[0] == k {
					vals = append(vals, v)
					index = index[1:]
					il--
				}
				return !(il > 0)
			})
		}
	}
	return
}

func (slce *Slice) Invoke(index int, method string, a ...interface{}) (result []interface{}) {
	if method != "" {
		if val := slce.Get(index); val != nil {
			_, result = reflection.ReflectCallMethod(val, method, a...)
		}
	}
	return
}

func (slce *Slice) Field(index int, field string, a ...interface{}) (result []interface{}) {
	if field != "" {
		if val := slce.Get(index); val != nil {
			_, result = reflection.ReflectCallField(val, field, a...)
		}
	}
	return
}

func (slce *Slice) Find(k ...interface{}) (value interface{}, found bool) {
	if len(k) == 0 {
		if ks, _ := k[0].(string); ks != "" && strings.Contains(ks, ",") {
			ksarr := strings.Split(ks, ",")
			k = make([]interface{}, len(ksarr))
			for kn, kv := range ksarr {
				k[kn] = kv
			}
		}
	}
	found = findvalue(nil, slce, func(val interface{}) {
		value = val
	}, k...)
	return
}

func (slce *Slice) Dispose() {
	if slce != nil {
		if mp := slce.Map; mp != nil {
			slce.Map = nil
			vals := []interface{}{}
			mp.Range(func(k interface{}, v interface{}) (stop bool) {
				if vmp, _ := v.(*Map); vmp != nil {
					vals = append(vals, vmp)
				} else if vslce, _ := v.(*Slice); v != nil {
					vals = append(vals, vslce)
				}
				return
			})
			for _, v := range vals {
				if vmp, _ := v.(*Map); vmp != nil {
					vmp.Dispose()
				} else if vslce, _ := v.(*Slice); v != nil {
					vslce.Dispose()
				}
			}
			mp.Dispose()
			mp = nil
		}
		slce = nil
	}
}

func (slce *Slice) ForEach(eachitem func(interface{}, int, bool, bool) bool) {
	if slce != nil && eachitem != nil {
		if mp := slce.Map; mp != nil {
			first := true
			cnt := mp.Count()
			mp.Range(func(k interface{}, v interface{}) (stop bool) {
				stop = !eachitem(v, k.(int), first, cnt-1 == k)
				if first {
					first = false
				}
				return stop || cnt-1 == k
			})
		}
	}
}
