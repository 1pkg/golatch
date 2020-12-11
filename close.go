package golatch

import (
	"C"
	"fmt"
	"reflect"
)

// NotWritableChannel defines not writable channel error type.
type NotWritableChannel struct {
	Kind reflect.Kind
	Dir  *reflect.ChanDir
}

func (err NotWritableChannel) Error() string {
	if err.Dir != nil {
		return fmt.Sprintf("provided entity %q is not a writable channel", *err.Dir)
	}
	return fmt.Sprintf("provided entity kind %q is not a writable channel", err.Kind)
}

// ChannelTypeMismatch defines type mismatch between underlying channel and value kind error type.
type ChannelTypeMismatch struct {
	ValKind, ChKind reflect.Kind
}

func (err ChannelTypeMismatch) Error() string {
	return fmt.Sprintf("provided value kind %q doesn't match provided underlying channel kind %q", err.ValKind, err.ChKind)
}

// Cancel defines Close cancelation function type.
type Cancel func()

// Close idempotently closes provided chan and stores provided value
// to return as this channel closed value instead of empty value.
// If provided channel is not a writable channel `NotWritableChannel` error is returned.
// If provided value doesn't match underlying channel type `ChannelTypeMismatch` error is returned.
// To cancel the effect of closed value replace call cancel function.
// Note that provided value won't be automatically collected by GC together with provided channel
// to remove provided value from the storage call cancel function.
// Note that after cancelation is called next call to Close will cause a panic `close of closed channel`.
// Note that unless cancelation is called next call to Close is safe and won't cause any panic
// but just update storage value with new provided value.
// Note that in order to achieve such effect golatch uses package based on `bou.ke/monkey`
// to patch all existing channel receive entrypoints, so golatch inherits the same list of restrictions.
func Close(ch interface{}, val interface{}) (Cancel, error) {
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
