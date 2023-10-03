// Copyright 2023 Gravitational, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resume

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"os"
	"slices"
	"sync"
	"time"
)

const (
	bufferSize = 16 * 1024 * 1024
	maxFrame   = 16 * 1024
)

func NewConn(localAddr, remoteAddr net.Addr) *Conn {
	c := &Conn{
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		cond:       make(chan struct{}),
	}
	c.detachTimer = time.AfterFunc(30*time.Second, func() { c.Close() })
	return c
}

type Conn struct {
	detachTimer *time.Timer
	localAddr   net.Addr
	remoteAddr  net.Addr

	attached func()

	mu      sync.Mutex
	cond    chan struct{}
	waiters bool

	closed        bool
	readDeadline  time.Time
	writeDeadline time.Time

	// readPosition is the end of receiveBuffer
	readPosition  int64
	receiveBuffer []byte

	// replayPosition is the beginning of replayBuffer
	replayPosition int64
	replayBuffer   []byte
}

var _ net.Conn = (*Conn)(nil)

func (c *Conn) Attach(nc net.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.detachLocked()

	if c.closed {
		nc.Close()
		return
	}

	c.detachTimer.Stop()
	c.attached = sync.OnceFunc(func() { nc.Close() })
	c.broadcastLocked()
	go c.run(nc)
}

func (c *Conn) detachLocked() {
	for {
		if c.attached == nil {
			break
		}
		c.attached()
		c.waitLocked(nil, "detachLocked")
	}
}

func (c *Conn) Detach() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.detachLocked()
}

func (c *Conn) run(nc net.Conn) {
	defer nc.Close()
	defer func() {
		c.mu.Lock()
		c.detachTimer.Reset(30 * time.Second)
		c.attached = nil
		c.broadcastLocked()
		c.mu.Unlock()
	}()

	c.mu.Lock()
	readPositionOnce := c.readPosition
	sentWindowStart := c.readPosition - int64(len(c.receiveBuffer))
	c.mu.Unlock()

	ncR := bufio.NewReader(nc)

	buf := binary.AppendVarint(nil, readPositionOnce)
	buf = binary.AppendVarint(buf, bufferSize-(readPositionOnce-sentWindowStart))
	if _, err := nc.Write(buf); err != nil {
		return
	}

	remoteReadPosition, err := binary.ReadVarint(ncR)
	if err != nil {
		return
	}
	remoteWindowSize, err := binary.ReadVarint(ncR)
	if err != nil {
		return
	}

	c.mu.Lock()
	if remoteReadPosition < c.replayPosition || c.replayPosition+int64(len(c.replayBuffer)) < remoteReadPosition {
		c.mu.Unlock()
		return
	}
	c.replayBuffer = c.replayBuffer[remoteReadPosition-c.replayPosition:]
	c.replayPosition = remoteReadPosition
	c.broadcastLocked()
	c.mu.Unlock()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer nc.Close()

		for {
			advanceWindow, err := binary.ReadVarint(ncR)
			if err != nil || advanceWindow < 0 {
				return
			}
			frameSize, err := binary.ReadVarint(ncR)
			if err != nil || frameSize < 0 {
				return
			}

			c.mu.Lock()
			if advanceWindow > 0 {
				if advanceWindow > int64(len(c.replayBuffer)) {
					c.mu.Unlock()
					return
				}
				remoteWindowSize += advanceWindow
				c.replayBuffer = c.replayBuffer[advanceWindow:]
				c.replayPosition += advanceWindow
				c.broadcastLocked()
			}
			if frameSize == 0 {
				c.mu.Unlock()
				continue
			}
			if frameSize > bufferSize-int64(len(c.replayBuffer)) || frameSize > maxFrame {
				c.mu.Unlock()
				return
			}
			c.receiveBuffer = slices.Grow(c.receiveBuffer, int(frameSize))
			recvBuf := c.receiveBuffer[len(c.receiveBuffer):cap(c.receiveBuffer)][:frameSize]
			c.mu.Unlock()

			n, err := io.ReadFull(ncR, recvBuf)

			if n > 0 {
				c.mu.Lock()
				c.receiveBuffer = append(c.receiveBuffer, recvBuf[:n]...)
				c.readPosition += int64(n)
				c.broadcastLocked()
				c.mu.Unlock()
			}

			if err != nil {
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer nc.Close()

		for {
			var sendBuf []byte
			var sendAdvance int64

			c.mu.Lock()
			for {
				sendBuf, sendAdvance = nil, 0
				if remoteWindowSize > 0 && c.replayPosition+int64(len(c.replayBuffer)) > remoteReadPosition {
					sendBuf = c.replayBuffer[remoteReadPosition-c.replayPosition:]
					sendBuf = sendBuf[:min(remoteWindowSize, int64(len(sendBuf)))]
					sendBuf = sendBuf[:min(len(sendBuf), maxFrame)]
				}
				windowStart := c.readPosition - int64(len(c.receiveBuffer))
				if windowStart > sentWindowStart {
					sendAdvance = windowStart - sentWindowStart
				}
				if len(sendBuf) > 0 || sendAdvance > 0 {
					break
				}
				c.waitLocked(nil, "write sendBuf loop")
			}
			c.mu.Unlock()

			metaBuf := binary.AppendVarint(nil, sendAdvance)
			metaBuf = binary.AppendVarint(metaBuf, int64(len(sendBuf)))
			if _, err := nc.Write(metaBuf); err != nil {
				return
			}
			if _, err := nc.Write(sendBuf); err != nil {
				return
			}

			c.mu.Lock()
			sentWindowStart += sendAdvance
			remoteReadPosition += int64(len(sendBuf))
			remoteWindowSize -= int64(len(sendBuf))
			c.mu.Unlock()
		}
	}()

	wg.Wait()
}

// Close implements [net.Conn].
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.detachLocked()

	if c.closed {
		return nil
	}

	c.closed = true
	c.detachTimer.Stop()
	c.broadcastLocked()

	return nil
}

// LocalAddr implements [net.Conn].
func (c *Conn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr implements [net.Conn].
func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *Conn) broadcastLocked() {
	if !c.waiters {
		return
	}
	close(c.cond)
	c.cond = make(chan struct{})
}

func (c *Conn) waitLocked(timeoutC <-chan time.Time, name string) (timeout bool) {
	cond := c.cond
	c.waiters = true
	c.mu.Unlock()
	defer c.mu.Lock()
	select {
	case <-cond:
		return false
	case <-timeoutC:
		return true
	}
}

func deadlineTimer(deadline time.Time, timer *time.Timer) (*time.Timer, <-chan time.Time) {
	if deadline.IsZero() {
		return timer, nil
	}
	if timer == nil {
		timer = time.NewTimer(time.Until(deadline))
	} else {
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(time.Until(deadline))
	}
	return timer, timer.C
}

// SetDeadline implements [net.Conn].
func (c *Conn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return net.ErrClosed
	}

	c.readDeadline = t
	c.writeDeadline = t
	c.broadcastLocked()
	return nil
}

// SetReadDeadline implements [net.Conn].
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return net.ErrClosed
	}

	c.readDeadline = t
	c.broadcastLocked()
	return nil
}

// SetWriteDeadline implements [net.Conn].
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return net.ErrClosed
	}

	c.writeDeadline = t
	c.broadcastLocked()
	return nil
}

// Read implements [net.Conn].
func (c *Conn) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, net.ErrClosed
	}

	if !c.readDeadline.IsZero() && time.Now().After(c.readDeadline) {
		return 0, os.ErrDeadlineExceeded
	}

	if len(b) == 0 {
		return 0, nil
	}

	var deadlineT *time.Timer
	defer func() {
		if deadlineT != nil {
			deadlineT.Stop()
		}
	}()

	for {
		if len(c.receiveBuffer) > 0 {
			n := copy(b, c.receiveBuffer)
			c.receiveBuffer = c.receiveBuffer[n:]
			c.broadcastLocked()
			return n, nil
		}

		var deadlineC <-chan time.Time
		deadlineT, deadlineC = deadlineTimer(c.readDeadline, deadlineT)

		if timeout := c.waitLocked(deadlineC, "Read"); timeout {
			return 0, os.ErrDeadlineExceeded
		}

		if c.closed {
			return 0, net.ErrClosed
		}
	}
}

// Write implements [net.Conn].
func (c *Conn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, net.ErrClosed
	}

	if !c.writeDeadline.IsZero() && time.Now().After(c.writeDeadline) {
		return 0, os.ErrDeadlineExceeded
	}

	if len(b) == 0 {
		return 0, nil
	}

	var deadlineT *time.Timer
	defer func() {
		if deadlineT != nil {
			deadlineT.Stop()
		}
	}()

	for {
		if bufferSize > len(c.replayBuffer) {
			s := min(bufferSize-len(c.replayBuffer), len(b))
			c.replayBuffer = append(c.replayBuffer, b[:s]...)
			b = b[s:]
			n += s
			c.broadcastLocked()
		}

		if len(b) == 0 {
			return n, nil
		}

		var deadlineC <-chan time.Time
		deadlineT, deadlineC = deadlineTimer(c.writeDeadline, deadlineT)

		if timeout := c.waitLocked(deadlineC, "Write"); timeout {
			return n, os.ErrDeadlineExceeded
		}

		if c.closed {
			return n, net.ErrClosed
		}
	}
}
