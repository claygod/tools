# Chan Agent

The channel agent allows you to add messages to the channel based on priority.
If the message has a high priority, it will be delivered first in the queue, pushing the remaining messages.
You can also clear the message queue (by calling a separate method or directly when sending a new message).

## Install

Copy the file for your version of the Golang to the "src/runtime" directory.

### Golang versions

- [x] 1.10.3
- [x] 1.10.5
- [x] 1.11.2
- [ ] other versions will do on request (Issue)

## Usage

### Priority

```go
package main

import (
	"fmt"
)

func main(){
	ch := make(chan int64, 7)
	ag := runtime.NewChanAgent(ch)
	
	ch <- 700
	
	var item int64 = 200
	ag.Send(unsafe.Pointer(&item), true, false)
	
	out1 := <-ch
	fmt.Println(out1) // "200"
	out2 := <-ch
	fmt.Println(out2) // "700"
}
```

### Send and clean

```go
package main

import (
	"fmt"
)

func main(){
	ch := make(chan int64, 7)
	ag := runtime.NewChanAgent(ch)
	
	ch <- 700
	
	var item int64 = 200
	ag.Send(unsafe.Pointer(&item), true, true)
	
	fmt.Println(len(ch)) // "1"
	out := <-ch
	fmt.Println(out) // "200"
}
```

### Clean

```go
package main

import (
	"fmt"
)

func main(){
	ch := make(chan int64, 7)
	ag := runtime.NewChanAgent(ch)
	
	fmt.Println(len(ch)) // "0"
	ch <- 700
	fmt.Println(len(ch)) // "1"

	ag.Clean()
	fmt.Println(len(ch)) // "0"
	
	var item int64 = 200
	ag.Send(unsafe.Pointer(&item), true, true)
}
```
	

## ToDo

- [x] Priority when sending to channel
- [x] Cleaning the buffer channel
- [ ] More priority levels

Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>