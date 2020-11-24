package go2close

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClose2Types(t *testing.T) {
	chInt := make(chan int)
	_, errInt := Close2(chInt, 10)
	require.NoError(t, errInt)
	vInt, okInt := <-chInt
	require.EqualValues(t, 10, vInt)
	require.EqualValues(t, false, okInt)

	chUInt := make(chan uint)
	_, errUInt := Close2(chUInt, uint(10))
	require.NoError(t, errUInt)
	vUInt, okUInt := <-chUInt
	require.EqualValues(t, uint(10), vUInt)
	require.EqualValues(t, false, okUInt)

	chStr := make(chan string)
	_, errStr := Close2(chStr, "foobar")
	require.NoError(t, errStr)
	vStr, okStr := <-chStr
	require.EqualValues(t, "foobar", vStr)
	require.EqualValues(t, false, okStr)

	chSlice := make(chan []interface{})
	_, errSlice := Close2(chSlice, []interface{}{10, "foobar", &[]int{1, 2, 3}})
	require.NoError(t, errSlice)
	vSlice, okSlice := <-chSlice
	require.EqualValues(t, []interface{}{10, "foobar", &[]int{1, 2, 3}}, vSlice)
	require.EqualValues(t, false, okSlice)

	chMap := make(chan map[string]uint64)
	_, errMap := Close2(chMap, map[string]uint64{"foo": 1, "bar": 2})
	require.NoError(t, errMap)
	vMap, okMap := <-chMap
	require.EqualValues(t, map[string]uint64{"foo": 1, "bar": 2}, vMap)
	require.EqualValues(t, false, okMap)

	ch := make(chan int)
	chCh := make(chan chan int)
	_, errCh := Close2(chCh, ch)
	require.NoError(t, errCh)
	vCh, okCh := <-chCh
	require.EqualValues(t, ch, vCh)
	require.EqualValues(t, false, okCh)

	type complex struct {
		a int
		b string
		c *complex
	}
	var cmplx complex

	chStruct := make(chan complex)
	_, errStruct := Close2(chStruct, complex{a: 1, c: &cmplx})
	require.NoError(t, errStruct)
	vStruct, okStruct := <-chStruct
	require.EqualValues(t, complex{a: 1, c: &cmplx}, vStruct)
	require.EqualValues(t, false, okStruct)

	vcmplx := &complex{b: "val", c: &cmplx}
	chPtr := make(chan *complex)
	_, errPtr := Close2(chPtr, vcmplx)
	require.NoError(t, errPtr)
	vPtr, okPtr := <-chPtr
	require.EqualValues(t, vcmplx, vPtr)
	require.EqualValues(t, false, okPtr)
}

func TestClose2Send(t *testing.T) {
	ch := make(chan int)
	_, err := Close2(ch, 10)
	require.NoError(t, err)
	require.Panics(t, func() {
		ch <- 1
	})
}

func TestClose2Invariant(t *testing.T) {
	_, err := Close2(10, 100)
	require.EqualValues(t, NotWritableChannel{Kind: reflect.Int}, err)
	require.EqualValues(t, `provided entity kind "int" is not a writable channel`, err.Error())

	func(ch <-chan int) {
		_, err := Close2(ch, 100)
		dir := reflect.RecvDir
		require.EqualValues(t, NotWritableChannel{Kind: reflect.Chan, Dir: &dir}, err)
		require.EqualValues(t, `provided entity "<-chan" is not a writable channel`, err.Error())
	}(make(chan int))

	func(ch chan<- int) {
		_, err := Close2(ch, 100)
		require.NoError(t, err)
	}(make(chan int))

	ch := make(chan int)
	_, err = Close2(ch, "bbbb")
	require.EqualValues(t, ChannelTypeMismatch{ValKind: reflect.String, ChKind: reflect.Int}, err)
	require.EqualValues(t, `provided value kind "string" doesn't match provided channel kind "int"`, err.Error())
}

func TestClose2Idempotence(t *testing.T) {
	ch := make(chan int)
	_, err := Close2(ch, 10)
	require.NoError(t, err)
	v, ok := <-ch
	require.EqualValues(t, 10, v)
	require.EqualValues(t, false, ok)

	_, err = Close2(ch, 10)
	require.NoError(t, err)
	v, ok = <-ch
	require.EqualValues(t, 10, v)
	require.EqualValues(t, false, ok)

	del, err := Close2(ch, 15)
	require.NoError(t, err)
	v, ok = <-ch
	require.EqualValues(t, 15, v)
	require.EqualValues(t, false, ok)
	del()
	v, ok = <-ch
	require.EqualValues(t, 0, v)
	require.EqualValues(t, false, ok)
}

func TestClose2Multi(t *testing.T) {
	ch1 := make(chan int, 2)
	ch2 := make(chan int, 2)
	ch3 := make(chan int, 2)
	ch1 <- 1
	ch2 <- 2
	ch3 <- 3

	_, err := Close2(ch1, 10)
	require.NoError(t, err)
	_, err = Close2(ch2, 20)
	require.NoError(t, err)
	_, err = Close2(ch3, 30)
	require.NoError(t, err)

	v1, ok1 := <-ch1
	require.EqualValues(t, 1, v1)
	require.EqualValues(t, true, ok1)
	v1, ok1 = <-ch1
	require.EqualValues(t, 10, v1)
	require.EqualValues(t, false, ok1)
	v1, ok1 = <-ch1
	require.EqualValues(t, 10, v1)
	require.EqualValues(t, false, ok1)

	v2, ok2 := <-ch2
	require.EqualValues(t, 2, v2)
	require.EqualValues(t, true, ok2)
	v2, ok2 = <-ch2
	require.EqualValues(t, 20, v2)
	require.EqualValues(t, false, ok2)
	v2, ok2 = <-ch2
	require.EqualValues(t, 20, v2)
	require.EqualValues(t, false, ok2)

	v3, ok3 := <-ch3
	require.EqualValues(t, 3, v3)
	require.EqualValues(t, true, ok3)
	v3, ok3 = <-ch3
	require.EqualValues(t, 30, v3)
	require.EqualValues(t, false, ok3)
	v3, ok3 = <-ch3
	require.EqualValues(t, 30, v3)
	require.EqualValues(t, false, ok3)
}

func TestClose2Select(t *testing.T) {
	ch := make(chan int)
	del, err := Close2(ch, 10)
	require.NoError(t, err)
	select {
	case v, ok := <-ch:
		require.EqualValues(t, 10, v)
		require.EqualValues(t, false, ok)
	default:
		require.True(t, false)
	}
	del()
	select {
	case v, ok := <-ch:
		require.EqualValues(t, 0, v)
		require.EqualValues(t, false, ok)
	default:
		require.True(t, false)
	}
}

func TestClose2Reflect(t *testing.T) {
	ch := make(chan int)
	del, err := Close2(ch, 10)
	require.NoError(t, err)
	chRef := reflect.ValueOf(ch)
	v, ok := chRef.Recv()
	require.EqualValues(t, 10, v.Interface())
	require.EqualValues(t, false, ok)
	del()
	v, ok = chRef.Recv()
	require.EqualValues(t, 0, v.Interface())
	require.EqualValues(t, false, ok)
}
