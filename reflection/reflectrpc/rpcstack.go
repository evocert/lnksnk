package reflectrpc

import (
	"fmt"
	"strings"
	"sync"
)

type RpcStack struct {
	refmap *sync.Map
	objmap *sync.Map
}

func NewRpcStack() (rpcstck *RpcStack) {
	rpcstck = &RpcStack{refmap: &sync.Map{}, objmap: &sync.Map{}}
	return
}

func (rcpstck *RpcStack) Register(aliasref string, objref interface{}) (didreg bool) {
	if rcpstck != nil && aliasref != "" && objref != nil {
		if pntraddr := fmt.Sprintf("%v", &objref); pntraddr != "" {
			if refmap, objmap := rcpstck.refmap, rcpstck.objmap; refmap != nil && objmap != nil {
				if _, objrefok := objmap.Load(pntraddr); !objrefok {
					objmap.Store(pntraddr, objref)
				}
				if ref, refok := refmap.Load(aliasref); !refok || (refok && ref != pntraddr) {
					if didreg = (refok && ref == pntraddr); didreg {
						return
					}
					refmap.Store(aliasref, pntraddr)
					didreg = true
				}
			}
		}
	}
	return didreg
}

func (rcpstck *RpcStack) RpcItem(ref string) (rpctitm *rpcitem) {
	if rcpstck != nil && ref != "" {
		if strings.HasPrefix(ref, "/") {
			if ref = ref[1:]; ref == "" {
				return
			}
		}
		if refmap, objmap := rcpstck.refmap, rcpstck.objmap; refmap != nil && objmap != nil {
			if pntref, ptnrrefok := refmap.Load(ref); ptnrrefok {
				if obj, objok := objmap.Load(pntref); objok {
					rpctitm = &rpcitem{objref: obj, ObjAddr: pntref.(string)}
				}
			} else {
				if obj, objok := objmap.Load(ref); objok {
					rpctitm = &rpcitem{objref: obj, ObjAddr: ref}
				}
			}
		}
	}
	return
}
