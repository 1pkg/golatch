package golock

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloseTypes(t *testing.T) {
	chInt := make(chan int)
	_, errInt := Close(chInt, 10)
	require.NoError(t, errInt)
	vInt, okInt := <-chInt
	require.EqualValues(t, 10, vInt)
	require.EqualValues(t, false, okInt)

	chUInt := make(chan uint)
	_, errUInt := Close(chUInt, uint(10))
	require.NoError(t, errUInt)
	vUInt, okUInt := <-chUInt
	require.EqualValues(t, uint(10), vUInt)
	require.EqualValues(t, false, okUInt)

	chStr := make(chan string)
	_, errStr := Close(chStr, "foobar")
	require.NoError(t, errStr)
	vStr, okStr := <-chStr
	require.EqualValues(t, "foobar", vStr)
	require.EqualValues(t, false, okStr)

	chSlice := make(chan []interface{})
	_, errSlice := Close(chSlice, []interface{}{10, "foobar", &[]int{1, 2, 3}})
	require.NoError(t, errSlice)
	vSlice, okSlice := <-chSlice
	require.EqualValues(t, []interface{}{10, "foobar", &[]int{1, 2, 3}}, vSlice)
	require.EqualValues(t, false, okSlice)

	chMap := make(chan map[string]uint64)
	_, errMap := Close(chMap, map[string]uint64{"foo": 1, "bar": 2})
	require.NoError(t, errMap)
	vMap, okMap := <-chMap
	require.EqualValues(t, map[string]uint64{"foo": 1, "bar": 2}, vMap)
	require.EqualValues(t, false, okMap)

	ch := make(chan int)
	chCh := make(chan chan int)
	_, errCh := Close(chCh, ch)
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
	_, errStruct := Close(chStruct, complex{a: 1, c: &cmplx})
	require.NoError(t, errStruct)
	vStruct, okStruct := <-chStruct
	require.EqualValues(t, complex{a: 1, c: &cmplx}, vStruct)
	require.EqualValues(t, false, okStruct)

	vcmplx := &complex{b: "val", c: &cmplx}
	chPtr := make(chan *complex)
	_, errPtr := Close(chPtr, vcmplx)
	require.NoError(t, errPtr)
	vPtr, okPtr := <-chPtr
	require.EqualValues(t, vcmplx, vPtr)
	require.EqualValues(t, false, okPtr)
}

func TestCloseSend(t *testing.T) {
	ch := make(chan int)
	_, err := Close(ch, 10)
	require.NoError(t, err)
	require.Panics(t, func() {
		ch <- 1
	})
}

func TestCloseInvariant(t *testing.T) {
	_, err := Close(10, 100)
	require.EqualValues(t, NotWritableChannel{Kind: reflect.Int}, err)
	require.EqualValues(t, `provided entity kind "int" is not a writable channel`, err.Error())

	func(ch <-chan int) {
		_, err := Close(ch, 100)
		dir := reflect.RecvDir
		require.EqualValues(t, NotWritableChannel{Kind: reflect.Chan, Dir: &dir}, err)
		require.EqualValues(t, `provided entity "<-chan" is not a writable channel`, err.Error())
	}(make(chan int))

	func(ch chan<- int) {
		_, err := Close(ch, 100)
		require.NoError(t, err)
	}(make(chan int))

	ch := make(chan int)
	_, err = Close(ch, "bbbb")
	require.EqualValues(t, ChannelTypeMismatch{ValKind: reflect.String, ChKind: reflect.Int}, err)
	require.EqualValues(t, `provided value kind "string" doesn't match provided underlying channel kind "int"`, err.Error())
}

func TestCloseIdempotence(t *testing.T) {
	ch := make(chan int)
	_, err := Close(ch, 10)
	require.NoError(t, err)
	v, ok := <-ch
	require.EqualValues(t, 10, v)
	require.EqualValues(t, false, ok)

	_, err = Close(ch, 10)
	require.NoError(t, err)
	v, ok = <-ch
	require.EqualValues(t, 10, v)
	require.EqualValues(t, false, ok)

	del, err := Close(ch, 15)
	require.NoError(t, err)
	v, ok = <-ch
	require.EqualValues(t, 15, v)
	require.EqualValues(t, false, ok)
	del()
	v, ok = <-ch
	require.EqualValues(t, 0, v)
	require.EqualValues(t, false, ok)
}

func TestCloseMulti(t *testing.T) {
	ch1 := make(chan int, 2)
	ch2 := make(chan int, 2)
	ch3 := make(chan int, 2)
	ch1 <- 1
	ch2 <- 2
	ch3 <- 3

	_, err := Close(ch1, 10)
	require.NoError(t, err)
	_, err = Close(ch2, 20)
	require.NoError(t, err)
	_, err = Close(ch3, 30)
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

func TestCloseSelect(t *testing.T) {
	ch := make(chan int)
	del, err := Close(ch, 10)
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

func TestCloseReflect(t *testing.T) {
	ch := make(chan int)
	del, err := Close(ch, 10)
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
