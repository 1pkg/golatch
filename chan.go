package golatch

import (
	"C"
	"sync"
	"unsafe"

	"github.com/1pkg/gomonkey"
)

// tp from `runtime._type`
type tp struct {
	_    uintptr
	_    uintptr
	_    uint32
	_    uint8
	_    uint8
	_    uint8
	kind uint8
}

// hchan from `runtime.chan`
type hchan struct {
	_        uint
	_        uint
	_        unsafe.Pointer
	_        uint16
	closed   uint32
	elemtype *tp
}

// iface from `runtime.type`
type iface struct {
	_   *tp
	val unsafe.Pointer
}

// scase from `runtime.scase`
type scase struct {
	ch   *hchan
	elem unsafe.Pointer
}

//go:linkname chanrecv runtime.chanrecv
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool)

//go:linkname typedmemmove runtime.typedmemmove
func typedmemmove(tp *tp, dst, src unsafe.Pointer)

//go:linkname chanrecv1 runtime.chanrecv1
func chanrecv1(c *hchan, elem unsafe.Pointer)

//go:linkname chanrecv2 runtime.chanrecv2
func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool)

//go:linkname selectnbrecv runtime.selectnbrecv
func selectnbrecv(elem unsafe.Pointer, c *hchan) (selected bool)

//go:linkname selectnbrecv2 runtime.selectnbrecv2
func selectnbrecv2(elem unsafe.Pointer, received *bool, c *hchan) (selected bool)

//go:linkname reflectChanrecv reflect.chanrecv
func reflectChanrecv(c *hchan, nb bool, elem unsafe.Pointer) (selected bool, received bool)

//go:linkname selectgo runtime.selectgo
func selectgo(cas0 *scase, order0 *uint16, ncases int) (int, bool)

type chStore sync.Map

func (*chStore) deref(val interface{}, tp *tp) unsafe.Pointer {
	ifc := (*iface)(unsafe.Pointer(&val))
	// see `reflect.ifaceIndir` implementation of indirect check
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

func (s *chStore) load(key uintptr, tp *tp, dst unsafe.Pointer) {
	if val, ok := ((*sync.Map)(s)).Load(key); ok {
		// see `runtime.chanrecv` implementation of new message read
		if dst == nil {
			return
		}
		typedmemmove(tp, dst, s.deref(val, tp))
	}
}

func (s *chStore) proc(rec bool, ch *hchan, elem unsafe.Pointer) {
	// procces only if chan is closed and drained
	if ch != nil && ch.closed == 1 && !rec {
		ptr := uintptr(unsafe.Pointer(ch))
		s.load(ptr, ch.elemtype, elem)
	}
}

// init patches all existing chan receive entrypoints
// - direct chan receive
// - chan select statement
// - reflect chan receive
// Note that multiselect statement is patched via patch guard,
// which makes it not thread safe.
func init() {
	gomonkey.Patch(chanrecv1, func(ch *hchan, elem unsafe.Pointer) {
		_, rec := chanrecv(ch, elem, true)
		gchStore.proc(rec, ch, elem)
	})
	gomonkey.Patch(chanrecv2, func(ch *hchan, elem unsafe.Pointer) bool {
		_, rec := chanrecv(ch, elem, true)
		gchStore.proc(rec, ch, elem)
		return rec
	})
	gomonkey.Patch(selectnbrecv, func(elem unsafe.Pointer, ch *hchan) bool {
		sel, rec := chanrecv(ch, elem, false)
		gchStore.proc(rec, ch, elem)
		return sel
	})
	gomonkey.Patch(selectnbrecv2, func(elem unsafe.Pointer, recv *bool, ch *hchan) bool {
		sel, rec := chanrecv(ch, elem, false)
		gchStore.proc(rec, ch, elem)
		*recv = rec
		return sel
	})
	gomonkey.Patch(reflectChanrecv, func(ch *hchan, nb bool, elem unsafe.Pointer) (bool, bool) {
		sel, rec := chanrecv(ch, elem, !nb)
		gchStore.proc(rec, ch, elem)
		return sel, rec
	})
	var g *gomonkey.PatchGuard
	g = gomonkey.Patch(selectgo, func(cas0 *scase, order0 *uint16, ncases int) (int, bool) {
		g.Unpatch()
		defer g.Restore()
		idx, rec := selectgo(cas0, order0, ncases)
		// NOTE: In order to maintain a lean stack size, the number of scases
		// is capped at 65536.
		cas1 := (*[1 << 16]scase)(unsafe.Pointer(cas0))
		scases := cas1[:ncases:ncases]
		obj := scases[idx]
		gchStore.proc(rec, obj.ch, obj.elem)
		return idx, rec
	})
}

var gchStore chStore
