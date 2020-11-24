# Go2Close 🔒

[![lint](https://github.com/1pkg/go2close/workflows/lint/badge.svg)](https://github.com/1pkg/go2close/actions?query=workflow%3Alint+branch%3Amaster+)
[![test](https://github.com/1pkg/go2close/workflows/test/badge.svg)](https://github.com/1pkg/go2close/actions?query=workflow%3Atest+branch%3Amaster+)
[![report](https://goreportcard.com/badge/github.com/1pkg/go2close?cache=1)](https://goreportcard.com/report/github.com/1pkg/go2close)
[![version](https://img.shields.io/github/go-mod/go-version/1pkg/go2close?nocache=1)](https://github.com/1pkg/go2close/blob/master/go.mod)
[![license](https://img.shields.io/github/license/1pkg/go2close?cache=1)](LICENSE)
[![godoc](https://img.shields.io/badge/godoc-godoc-green)](https://pkg.go.dev/github.com/1pkg/go2close?tab=doc)

`GOPROXY=direct go get -u github.com/1pkg/go2close`

## Introduction

Go2Close patches go runtime to provides a way to close a chan idempotently + overwrite empty value returned from that closed channel.

<table>
<tr>
<th>With Go2Close</th>
<th>Without Go2Close</th>
</tr>

<tr>
<td>

```go
package main

import (
	"fmt"
	"sync"
	"github.com/1pkg/go2close"
)

func main() {
	ch := make(chan int)
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go worker(i, ch, &wg)
	}
	go2close.Close2(ch, 10)
	wg.Wait()
}

func worker(i int, ch chan int, wg *sync.WaitGroup) {
	v, ok := <- ch // 10, false
	if ok || v != 10 {
		panic("unreachable") // won't panic
	}
	fmt.Printf("worker %d chan is closed with value %d\n", i, v)
	wg.Done()
}
```

</td>
<td>

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	ch := make(chan int)
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go worker(i, ch, &wg)
	}
	close(ch)
	wg.Wait()
}

func worker(i int, ch chan int, wg *sync.WaitGroup) {
	v, ok := <- ch // 0, false
	if ok || v != 10 {
		panic("unreachable") // will panic
	}
	fmt.Printf("worker %d chan is closed with value %d\n", i, v)
	wg.Done()
}
```

</td>
</tr>
</table>

## Internals

Go2Close exposes single function `Close2` that idempotently closes provided chan and stores provided value to return as this channel closed value instead of empty value. If provided channel is not a writable channel `NotWritableChannel` error is returned. If provided value doesn't match underlying channel type `ChannelTypeMismatch` error is returned.  
To cancel the effect of closed value replace call cancel function. Note that provided value won't be automatically collected by GC together with provided channel to remove provided value from the storage call cancel function. Note that after cancelation is called next call to `Close2` will cause a panic `close of closed channel`. Note that unless cancelation is called next call to `Close2` is safe and won't cause any panic but just update storage value with new provided value. Note that in order to achieve such effect go2close uses [bou.ke/monkey](https://github.com/bouk/monkey) package to patch all existing channel receive entrypoints:

- direct chan receive
- chan select statement
- reflect chan receive

This makes go2close inherits the same list of restrictions as `bou.ke/monkey` [has](https://github.com/bouk/monkey#notes).

## Licence

Go2Close is licensed under the MIT License.  
See [LICENSE](LICENSE) for the full license text.
