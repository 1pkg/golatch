package main

import (
	"C"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"bou.ke/monkey"
)

type _type struct{}

type hchan struct {
	_        uint
	_        uint
	_        unsafe.Pointer
	_        uint16
	closed   uint32
	elemtype *_type
}

type iface struct {
	_   unsafe.Pointer
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

//go:linkname typedmemclr runtime.typedmemmove
func typedmemclr(typ *_type, ptr unsafe.Pointer)

type chStore sync.Map

func (s *chStore) push(key uintptr, val interface{}) {
	((*sync.Map)(s)).Store(key, val)
}

func (s *chStore) load(key uintptr, tp *_type, dst unsafe.Pointer) {
	if val, ok := ((*sync.Map)(s)).Load(key); ok {
		vptr := (*iface)(unsafe.Pointer(&val)).val
		if dst == nil {
			typedmemclr(tp, dst)
		}
		typedmemmove(tp, dst, vptr)
	}
}

var gstore chStore

func Close(ch interface{}, val interface{}) error {
	chRef := reflect.ValueOf(ch)
	chTyp := chRef.Type()
	if chTyp.Kind() != reflect.Chan || chTyp.ChanDir()&reflect.SendDir == 0 {
		return fmt.Errorf("provided entity type %q is not a writable channel", chTyp.Kind())
	}
	if valTp := reflect.ValueOf(val).Type(); valTp.Kind() != chTyp.Elem().Kind() {
		return fmt.Errorf("provided value type %q doesn't match provided channel type %q", valTp.Kind(), chTyp.Elem().Kind())
	}
	gstore.push(chRef.Pointer(), val)
	chRef.Close()
	return nil
}

func load(rec bool, ch *hchan, elem unsafe.Pointer) {
	if !rec && ch.closed == 1 {
		ptr := uintptr(unsafe.Pointer(ch))
		gstore.load(ptr, ch.elemtype, elem)
	}
}

func init() {
	monkey.Patch(chanrecv1, func(ch *hchan, elem unsafe.Pointer) {
		_, rec := chanrecv(ch, elem, true)
		load(rec, ch, elem)
	})
	monkey.Patch(chanrecv2, func(ch *hchan, elem unsafe.Pointer) bool {
		_, rec := chanrecv(ch, elem, true)
		load(rec, ch, elem)
		return rec
	})
}

func main() {
	ch := make(chan []int, 2)
	ch <- []int{1, 2, 3}
	fmt.Println(Close(ch, []int{5}))
	v, ok := <-ch
	fmt.Println("RESULT", v, ok)
	v, ok = <-ch
	fmt.Println("RESULT", v, ok)
}
