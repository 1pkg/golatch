package go2close

import (
	"C"
	"sync"
	"unsafe"

	"bou.ke/monkey"
)

type _type struct {
	_    uintptr
	_    uintptr
	_    uint32
	_    uint8
	_    uint8
	_    uint8
	kind uint8
}

type hchan struct {
	_        uint
	_        uint
	_        unsafe.Pointer
	_        uint16
	closed   uint32
	elemtype *_type
}

type iface struct {
	_   *_type
	val unsafe.Pointer
}

//go:linkname chanrecv1 runtime.chanrecv1
func chanrecv1(c *hchan, elem unsafe.Pointer)

//go:linkname chanrecv2 runtime.chanrecv2
func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool)

//go:linkname chanrecv runtime.chanrecv
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool)

//go:linkname typedmemmove runtime.typedmemmove
func typedmemmove(tp *_type, dst, src unsafe.Pointer)

type chStore sync.Map

func (*chStore) deref(val interface{}, tp *_type) unsafe.Pointer {
	ifc := (*iface)(unsafe.Pointer(&val))
	if tp.kind&32 == 0 {
		return ifc.val
	}
	return unsafe.Pointer(&ifc.val)
}

func (s *chStore) has(key uintptr) bool {
	_, ok := ((*sync.Map)(s)).Load(key)
	return ok
}

func (s *chStore) del(key uintptr) {
	((*sync.Map)(s)).Delete(key)
}

func (s *chStore) push(key uintptr, val interface{}) {
	((*sync.Map)(s)).Store(key, val)
}

func (s *chStore) load(key uintptr, tp *_type, dst unsafe.Pointer) {
	if val, ok := ((*sync.Map)(s)).Load(key); ok {
		if dst == nil {
			return
		}
		typedmemmove(tp, dst, s.deref(val, tp))
	}
}

func (s *chStore) proc(rec bool, ch *hchan, elem unsafe.Pointer) {
	if !rec && ch.closed == 1 {
		ptr := uintptr(unsafe.Pointer(ch))
		s.load(ptr, ch.elemtype, elem)
	}
}

func init() {
	monkey.Patch(chanrecv1, func(ch *hchan, elem unsafe.Pointer) {
		_, rec := chanrecv(ch, elem, true)
		gchStore.proc(rec, ch, elem)
	})
	monkey.Patch(chanrecv2, func(ch *hchan, elem unsafe.Pointer) bool {
		_, rec := chanrecv(ch, elem, true)
		gchStore.proc(rec, ch, elem)
		return rec
	})
}

var gchStore chStore
