package main

import (
	"C"
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"bou.ke/monkey"
)

type hchan struct {
	_      uint
	_      uint
	_      unsafe.Pointer
	_      uint16
	closed uint32
}

type eface struct {
	typ, val unsafe.Pointer
}

//go:linkname chanrecv1 runtime.chanrecv1
func chanrecv1(c *hchan, elem unsafe.Pointer)

//go:linkname chanrecv2 runtime.chanrecv2
func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool)

//go:linkname chanrecv runtime.chanrecv
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool)

//go:linkname memmove runtime.memmove
func memmove(dst, src unsafe.Pointer, size int)

var cvalues sync.Map

func Close(ch interface{}, val interface{}) error {
	chRef := reflect.ValueOf(ch)
	chTyp := chRef.Type()
	if chTyp.Kind() != reflect.Chan || chTyp.ChanDir()&reflect.SendDir == 0 {
		return fmt.Errorf("provided entity type %q is not a writable channel", chTyp.Kind())
	}
	if valTp := reflect.ValueOf(val).Type(); valTp.Kind() != chTyp.Elem().Kind() {
		return fmt.Errorf("provided value type %q doesn't match provided channel type %q", valTp.Kind(), chTyp.Elem().Kind())
	}
	cvalues.Store(chRef.Pointer(), val)
	chRef.Close()
	return nil
}

func cload(ch *hchan, elem unsafe.Pointer) {
	if ch.closed == 1 {
		ptr := uintptr(unsafe.Pointer(ch))
		if val, ok := cvalues.Load(ptr); ok {
			vptr := (*eface)(unsafe.Pointer(&val)).val
			memmove(elem, vptr, int(unsafe.Sizeof(vptr)))
		}
	}
}

func main() {
	monkey.Patch(chanrecv1, func(ch *hchan, elem unsafe.Pointer) {
		chanrecv(ch, elem, true)
		cload(ch, elem)
	})
	monkey.Patch(chanrecv2, func(ch *hchan, elem unsafe.Pointer) (received bool) {
		_, received = chanrecv(ch, elem, true)
		cload(ch, elem)
		return
	})
	ch := make(chan int, 2)
	go func() {
		select {
		case v, ok := <-ch:
			fmt.Println("FROM GOROUTINE", v, ok)
		}
	}()
	time.Sleep(time.Second)
	Close(ch, 23)
	v, ok := <-ch
	fmt.Println("RESULT", v, ok)
	v, ok = <-ch
	fmt.Println("RESULT", v, ok)
	v, ok = <-ch
	fmt.Println("RESULT", v, ok)
}
