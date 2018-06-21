/**
 * roundrobin.go - roundrobin balance impl
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"errors"

	"../core"
)


var addrMap = make(map[string]int)
/**
 * Roundrobin balancer
 */
type RoundrobinBalancer struct {

	/* Current backend position */
	current int
}

/**
 * Elect backend using roundrobin strategy
 */
func (b *RoundrobinBalancer) Elect(context core.Context, backends []*core.Backend) (*core.Backend, error) {

	b.current = addrMap[context.String()]

	if len(backends) == 0 {
		return nil, errors.New("Can't elect backend, Backends empty")
	}

	if b.current >= len(backends) {
		b.current = 0
	}

	backend := backends[b.current]
	b.current += 1
	addrMap[context.String()] = b.current

	return backend, nil

}
