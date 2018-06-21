/**
 * context.go - proxy context
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package core

import "net"

type Context interface {
	String() string
	Ip() net.IP
	Port() int
}

/*
 * Proxy udp context
 */
type UdpContext struct {

	/**
	 * Current client remote address
	 */
	RemoteAddr net.UDPAddr
}

func (u UdpContext) String() string {
	return u.RemoteAddr.String()
}

func (u UdpContext) Ip() net.IP {
	return u.RemoteAddr.IP
}

func (u UdpContext) Port() int {
	return u.RemoteAddr.Port
}
