/**
 * discovery.go - discovery
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package discovery

import (
	"../config"
	"../core"
	"../logging"
	"time"
)

/**
 * Registry of factory methods for Discoveries
 */
var registry = make(map[string]func(config.DiscoveryConfig) interface{})

/**
 * Initialize type registry
 */
func init() {
	registry["static"] = NewStaticDiscovery
}

/**
 * Create new Discovery based on strategy
 */
func New(strategy string, cfg config.DiscoveryConfig) *Discovery {
	return registry[strategy](cfg).(*Discovery)
}

/**
 * Fetch func for pullig backends
 */
type FetchFunc func(config.DiscoveryConfig) (*[]core.Backend, error)

/**
 * Options for pull discovery
 */
type DiscoveryOpts struct {
	RetryWaitDuration time.Duration
}

/**
 * Discovery
 */
type Discovery struct {

	/**
	 * Cached backends
	 */
	backends *[]core.Backend

	/**
	 * Function to fetch / discovery backends
	 */
	fetch FetchFunc

	/**
	 * Options for fetch
	 */
	opts DiscoveryOpts

	/**
	 * Discovery configuration
	 */
	cfg config.DiscoveryConfig

	/**
	 * Channel where to push newly discovered backends
	 */
	out chan ([]core.Backend)
}

/**
 * Pull / fetch backends loop
 */
func (this *Discovery) Start() {

	log := logging.For("discovery")

	this.out = make(chan []core.Backend)

	interval := 0

	go func() {
		for {
			backends, err := this.fetch(this.cfg)

			if err != nil {
				log.Fatal("Fetch backends error ", err)
				return
			}

			// cache
			this.backends = backends

			// out
			this.out <- *this.backends

			// exit gorouting if no cacheTtl
			// used for static discovery
			if interval == 0 {
				return
			}
		}
	}()
}

/**
 * Stop discovery
 */
func (this *Discovery) Stop() {
	// TODO: Add stopping function
}

/**
 * Returns backends channel
 */
func (this *Discovery) Discover() <-chan []core.Backend {
	return this.out
}
