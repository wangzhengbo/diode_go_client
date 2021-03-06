// Diode Network Client
// Copyright 2019 IoT Blockchain Technology Corporation LLC (IBTC)
// Licensed under the Diode License, Version 1.0
package rpc

import (
	"io"
	"net"
	"sync"
	"time"
)

// Tunnel is a multiplex net copier in diode
type Tunnel struct {
	closeCh     chan struct{}
	conna       net.Conn
	connb       net.Conn
	idleTimeout time.Duration
	bufferSize  int
	cd          sync.Once
}

// NewTunnel returns a newly created Tunnel
func NewTunnel(conna, connb net.Conn, idleTimeout time.Duration, bufferSize int) (tun *Tunnel) {
	tun = &Tunnel{
		conna:       conna,
		connb:       connb,
		idleTimeout: idleTimeout,
		bufferSize:  bufferSize,
		closeCh:     make(chan struct{}),
	}
	return
}

func isClosed(closedCh <-chan struct{}) bool {
	select {
	case <-closedCh:
		return true
	default:
		return false
	}
}

func (tun *Tunnel) netCopyWithoutTimeout(input, output net.Conn, bufferSize int) (err error) {
	buf := make([]byte, bufferSize)
	for {
		var count int
		var writed int
		if isClosed(tun.closeCh) {
			return
		}
		count, err = input.Read(buf)
		if count > 0 {
			if isClosed(tun.closeCh) {
				return
			}
			writed, err = output.Write(buf[:count])
			if err != nil {
				return
			}
			if writed == 0 {
				err = io.EOF
				return
			}
		}
		// if count == 0 {
		// 	err = io.EOF
		// 	return
		// }
		if err != nil {
			return
		}
	}
}

func (tun *Tunnel) netCopy(input, output net.Conn, timeout time.Duration, bufferSize int) (err error) {
	buf := make([]byte, bufferSize)
	for {
		var count int
		var writed int
		if isClosed(tun.closeCh) {
			return
		}
		input.SetReadDeadline(time.Now().Add(timeout))
		count, err = input.Read(buf)
		if count > 0 {
			if isClosed(tun.closeCh) {
				return
			}
			output.SetWriteDeadline(time.Now().Add(timeout))
			writed, err = output.Write(buf[:count])
			if err != nil {
				return
			}
			if writed == 0 {
				err = io.EOF
				return
			}
		}
		// if count == 0 {
		// 	err = io.EOF
		// 	return
		// }
		if err != nil {
			return
		}
	}
}

// Copy start to bridge connections
func (tun *Tunnel) Copy() bool {
	if isClosed(tun.closeCh) {
		return true
	}
	if tun.idleTimeout > 0 {
		go tun.netCopy(tun.conna, tun.connb, tun.idleTimeout, tun.bufferSize)
		tun.netCopy(tun.connb, tun.conna, tun.idleTimeout, tun.bufferSize)
	} else {
		go tun.netCopyWithoutTimeout(tun.conna, tun.connb, tun.bufferSize)
		tun.netCopyWithoutTimeout(tun.connb, tun.conna, tun.bufferSize)
	}
	tun.Close()
	return isClosed(tun.closeCh)
}

// Closed returns the Tunnel was closed
func (tun *Tunnel) Closed() bool {
	return isClosed(tun.closeCh)
}

// Close the Tunnel
func (tun *Tunnel) Close() (err error) {
	tun.cd.Do(func() {
		close(tun.closeCh)
		tun.conna.Close()
		tun.connb.Close()
	})
	return
}
