package runtime

// Channel Agent
// Copyright © 2017-2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"runtime/internal/atomic"
	"unsafe"
)

type ChanAgent struct {
	hchan *hchan
}

func NewChanAgent(ch interface{}) *ChanAgent {
	chd := *(*iface)(unsafe.Pointer(&ch))
	return &ChanAgent{
		hchan: *(**hchan)(unsafe.Pointer(&chd.data)),
	}
}

/*
Send - send to channel.
Arguments:
- pointer
- priority, if the channel buffer is not empty, the message will be delivered first
- cleaning, the old queue in the buffer will be reset
*/
func (c *ChanAgent) Send(ep unsafe.Pointer, flagPriority bool, flagClean bool) bool {
	return chansendPr(c.hchan, ep, true, getcallerpc(), flagPriority, flagClean)
}

/*
Clean - the queue in the buffer will be reset
*/
func (c *ChanAgent) Clean() {
	lock(&c.hchan.lock)
	defer unlock(&c.hchan.lock)
	if c.hchan.closed != 0 {
		panic(plainError("ChanAgent: send on closed channel"))
	}
	chanClean(c.hchan)
}

func chansendPr(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr, flagPriority bool, flagClean bool) bool { // , args ...bool
	if c == nil {
		if !block {
			return false
		}
		gopark(nil, nil, waitReasonChanSendNilChan, traceEvGoStop, 2)
		throw("unreachable")
	}

	if debugChan {
		print("chansend: chan=", c, "\n")
	}

	if raceenabled {
		racereadpc(c.raceaddr(), callerpc, funcPC(chansend))
	}

	if !block && c.closed == 0 && ((c.dataqsiz == 0 && c.recvq.first == nil) ||
		(c.dataqsiz > 0 && c.qcount == c.dataqsiz)) {
		c.qcount = 0
		return false
	}

	var t0 int64
	if blockprofilerate > 0 {
		t0 = cputicks()
	}

	lock(&c.lock)

	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}

	if sg := c.recvq.dequeue(); sg != nil {
		// Found a waiting receiver. We pass the value we want to send
		// directly to the receiver, bypassing the channel buffer (if any).
		send(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true
	}

	if c.qcount < c.dataqsiz {
		// Space is available in the channel buffer. Enqueue the element to send.
		//		var qp unsafe.Pointer
		if flagClean {
			chanClean(c)
		}
		qp := chanbufPr(c, c.sendx, flagPriority)
		if raceenabled {
			raceacquire(qp)
			racerelease(qp)
		}
		typedmemmove(c.elemtype, qp, ep)
		c.sendx++
		if c.sendx == c.dataqsiz {
			c.sendx = 0
		}
		c.qcount++
		unlock(&c.lock)
		return true
	}

	if !block {
		unlock(&c.lock)
		return false
	}

	// Block on the channel. Some receiver will complete our operation for us.
	gp := getg()
	mysg := acquireSudog()
	mysg.releasetime = 0
	if t0 != 0 {
		mysg.releasetime = -1
	}
	// No stack splits between assigning elem and enqueuing mysg
	// on gp.waiting where copystack can find it.
	mysg.elem = ep
	mysg.waitlink = nil
	mysg.g = gp
	mysg.isSelect = false
	mysg.c = c
	gp.waiting = mysg
	gp.param = nil
	c.sendq.enqueue(mysg)
	goparkunlock(&c.lock, waitReasonChanSend, traceEvGoBlockSend, 3)

	// someone woke us up.
	if mysg != gp.waiting {
		throw("G waiting list is corrupted")
	}
	gp.waiting = nil
	if gp.param == nil {
		if c.closed == 0 {
			throw("chansend: spurious wakeup")
		}
		panic(plainError("send on closed channel"))
	}
	gp.param = nil
	if mysg.releasetime > 0 {
		blockevent(mysg.releasetime-t0, 2)
	}
	mysg.c = nil
	releaseSudog(mysg)
	return true
}

func chanbufPr(c *hchan, i uint, flagPriority bool) unsafe.Pointer {
	if flagPriority {
		for u := i; u > 0; u-- {
			p1 := (*unsafe.Pointer)(unsafe.Pointer(uintptr(c.buf) + uintptr(u-1)*uintptr(c.elemsize)))
			p2 := (*unsafe.Pointer)(unsafe.Pointer(uintptr(c.buf) + uintptr(u)*uintptr(c.elemsize)))
			*p2 = *p1
		}
		return c.buf
	}
	return add(c.buf, uintptr(i)*uintptr(c.elemsize))
}

func chanClean(c *hchan) {
	c.qcount = 0
	c.sendx = 0
}
