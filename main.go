package main

import (
	"C"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"bou.ke/monkey"
)

type NotWritableChannel struct {
	Kind reflect.Kind
	Dir  *reflect.ChanDir
}

func (err NotWritableChannel) Error() string {
	if err.Dir != nil {
		return fmt.Sprintf("provided entity %q is not a writable channel", *err.Dir)
	}
	return fmt.Sprintf("provided entity %q is not a writable channel", err.Kind)
}

type ChannelTypeMismatch struct {
	ValKind, ChKind reflect.Kind
}

func (err ChannelTypeMismatch) Error() string {
	return fmt.Sprintf("provided value %q doesn't match provided channel %q", err.ChKind, err.ValKind)
}

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

var gchStore chStore

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

type Cancel func()

func Close2(ch interface{}, val interface{}) (Cancel, error) {
	chRef := reflect.ValueOf(ch)
	chTyp := chRef.Type()
	if chTyp.Kind() != reflect.Chan {
		return nil, NotWritableChannel{Kind: chTyp.Kind()}
	}
	if dir := chTyp.ChanDir(); dir&reflect.SendDir == 0 {
		return nil, NotWritableChannel{Kind: chTyp.Kind(), Dir: &dir}
	}
	if valTp := reflect.ValueOf(val).Type(); valTp.Kind() != chTyp.Elem().Kind() {
		return nil, ChannelTypeMismatch{ValKind: valTp.Kind(), ChKind: chTyp.Elem().Kind()}
	}
	key := chRef.Pointer()
	if !gchStore.has(key) {
		chRef.Close()
	}
	gchStore.push(key, val)
	return func() { gchStore.del(key) }, nil
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

func main() {
	ch := make(chan map[string]int, 2)
	ch <- map[string]int{"foo": 1}
	_, err := Close2(ch, map[string]int{"var": 1})
	fmt.Println(err)
	v, ok := <-ch
	fmt.Println("RESULT", v, ok)
	v, ok = <-ch
	fmt.Println("RESULT", v, ok)
}
