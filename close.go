package go2close

import (
	"C"
	"fmt"
	"reflect"
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
