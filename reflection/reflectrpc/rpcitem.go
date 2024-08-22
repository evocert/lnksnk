package reflectrpc

import "github.com/lnksnk/lnksnk/reflection"

type rpcitem struct {
	objref  interface{}
	ObjAddr string
	fields  []string
	methods []string
}

func (rpcitm *rpcitem) Invoke(mname string, args ...interface{}) (called bool, result []interface{}) {
	called, result = rpcInvoke(rpcitm, mname, args...)
	return
}

func rpcInvoke(rpcitm *rpcitem, mname string, args ...interface{}) (called bool, result []interface{}) {
	if rpcitm != nil {
		if objref := rpcitm.objref; objref != nil {
			called, result = reflection.ReflectCallMethod(objref, mname, args...)
		}
	}
	return
}

func (rpcitm *rpcitem) Fields(flds ...string) (called bool, fields []string) {
	called, fields = rpcFields(rpcitm, flds...)
	return
}

func rpcFields(rpcitm *rpcitem, flds ...string) (called bool, fields []string) {
	if rpcitm != nil {
		fldsl := len(rpcitm.fields)
		if fldsl == 0 {
			if objref := rpcitm.objref; objref != nil {
				if called, fields = reflection.ReflectFields(objref); called {
					rpcitm.fields = fields[:]
					fldsl = len(rpcitm.fields)
				}
			}
		}
		if fldsl > 0 {

		}
	}
	return
}

func (rpcitm *rpcitem) Methods(mthds ...string) (called bool, methods []string) {
	called, methods = rpcMethods(rpcitm, mthds...)
	return
}

func rpcMethods(rpcitm *rpcitem, mthds ...string) (called bool, methods []string) {
	if rpcitm != nil {
		mthdsl := len(rpcitm.methods)
		if mthdsl == 0 {
			if objref := rpcitm.objref; objref != nil {
				if called, methods = reflection.ReflectMethods(objref); called {
					rpcitm.methods = methods[:]
					mthdsl = len(rpcitm.methods)
				}
			}
		}
		if mthdsl > 0 {

		}
	}
	return
}

func (rpcitm *rpcitem) Field(fname string, args ...interface{}) (called bool, result []interface{}) {
	called, result = rpcField(rpcitm, fname, args...)
	return
}

func rpcField(rpcitm *rpcitem, fname string, args ...interface{}) (called bool, result []interface{}) {
	if rpcitm != nil {
		if objref := rpcitm.objref; objref != nil {
			called, result = reflection.ReflectCallField(objref, fname, args...)
		}
	}
	return
}
