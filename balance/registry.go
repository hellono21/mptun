/**
 * registry.go - balancers registry
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package balance

import (
	"reflect"
	"../core"
)

/**
 * Type registry of available Balancers
 */
var typeRegistry = make(map[string]reflect.Type)

/**
 * Initialize type registry
 */
func init() {
	typeRegistry["roundrobin"] = reflect.TypeOf(RoundrobinBalancer{})
	typeRegistry["iphash"] = reflect.TypeOf(IphashBalancer{})
}

/**
 * Create new Balancer based on balancing strategy
 * Wrap it in middlewares if needed
 */
func New(balance string) core.Balancer {
	balancer := reflect.New(typeRegistry[balance]).Elem().Addr().Interface().(core.Balancer)

	return balancer;
}
