package main

import (
	"C"
	"fmt"
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

//go:linkname chanrecv1 runtime.chanrecv1
func chanrecv1(c *hchan, elem unsafe.Pointer)

//go:linkname chanrecv2 runtime.chanrecv2
func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool)

//go:linkname chanrecv runtime.chanrecv
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool)

//go:linkname memmove runtime.memmove
func memmove(dst, src unsafe.Pointer, size int)

func main() {
	monkey.Patch(chanrecv1, func(c *hchan, elem unsafe.Pointer) {
		chanrecv(c, elem, true)
		if c.closed == 1 {
			val := 10
			memmove(elem, unsafe.Pointer(&val), 8)
		}
	})
	monkey.Patch(chanrecv2, func(c *hchan, elem unsafe.Pointer) (received bool) {
		_, received = chanrecv(c, elem, true)
		if c.closed == 1 {
			val := 10
			memmove(elem, unsafe.Pointer(&val), 8)
		}
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
	close(ch)
	v, ok := <-ch
	fmt.Println("RESULT", v, ok)
	v, ok = <-ch
	fmt.Println("RESULT", v, ok)
	v, ok = <-ch
	fmt.Println("RESULT", v, ok)
}
