package scheduler

import (
	"time"

	"../logging"
	"../core"
	"../discovery"
	"../healthcheck"
)

type ElectRequest struct {
	Context		core.Context
	Response	chan core.Backend
	Err		chan error
}


type Scheduler struct {
	Balancer core.Balancer

	Healthcheck *healthcheck.Healthcheck

	Discovery *discovery.Discovery
	backends map[core.Target]*core.Backend
	backendsList []*core.Backend

	stopChan chan bool

	electChan chan ElectRequest
	LiveBackendsChan chan []core.Backend
}

func (this *Scheduler) Start() {
	log := logging.For("scheduler")
	log.Info("Starting scheduler")

	this.Discovery.Start()
	this.Healthcheck.Start()

	this.electChan = make(chan ElectRequest)
	this.stopChan = make (chan bool)
	this.LiveBackendsChan = make (chan []core.Backend)
	backendsPushTicker := time.NewTicker( 5* time.Second)

	go func() {
		for {
			select {
			case backends := <-this.Discovery.Discover():
				this.HandleBackendsUpdate(backends)
				this.Healthcheck.In <- this.Targets()
			case checkResult := <-this.Healthcheck.Out:
				this.HandleBackendLiveChange(checkResult.Target, checkResult.Live, checkResult.Rtt, checkResult.Loss)
			case electReq := <-this.electChan:
				this.HandleBackendElect(electReq)
			case <-backendsPushTicker.C:
				this.LiveBackendsChan <- this.LiveBackends()
			case <- this.stopChan:
				log.Info("Stopping scheduler")
				backendsPushTicker.Stop()
				this.Discovery.Stop()
				this.Healthcheck.Stop()
				return
			}
		}
	}()
}

func (this *Scheduler) Stop() {
	this.stopChan <- true
}

func (this *Scheduler) HandleBackendsUpdate(backends []core.Backend) {
	updated := map[core.Target]*core.Backend{}
	updatedList := make([]*core.Backend, len(backends))

	for i:= range backends {
		b := backends[i]
		oldB, ok := this.backends[b.Target]

		if ok {
			updatedB := oldB.MergeFrom(b)
			updated[oldB.Target] = updatedB
			updatedList[i] = updatedB
		} else {
			updated[b.Target] = &b
			updatedList[i] = &b
		}
	}

	this.backends = updated
	this.backendsList = updatedList
}

func (this *Scheduler) TakeBackend(context core.Context) (*core.Backend, error) {
	r := ElectRequest{
		context,
		make(chan core.Backend),
		make(chan error),
	}
	this.electChan <- r

	select {
	case err := <-r.Err:
		return nil, err
	case backend := <-r.Response:
		return &backend, nil
	}
}

func (this *Scheduler) HandleBackendElect(req ElectRequest) {
	var backends []*core.Backend

	for _, b := range this.backendsList {
		if !b.Stats.Live {
			continue
		}
		backends = append(backends, b)
	}

	backend, err := this.Balancer.Elect(req.Context, backends)
	if err != nil {
		req.Err <- err
		return
	}
	req.Response <- *backend
}

/**
 * Returns targets of current backends
 */
func (this *Scheduler) Targets() []core.Target {

	keys := make([]core.Target, 0, len(this.backends))
	for k := range this.backends {
		keys = append(keys, k)
	}

	return keys
}

/**
 * Return current backends
 */
func (this *Scheduler) Backends() []core.Backend {

	backends := make([]core.Backend, 0, len(this.backends))
	for _, b := range this.backends {
		backends = append(backends, *b)
	}

	return backends
}

/**
 * Return current live backends
 */
func (this *Scheduler) LiveBackends() []core.Backend {
	var backends []core.Backend

	for _, b := range this.backends {
		if !b.Stats.Live {
			continue
		}
		backends = append(backends, *b)
	}

	return backends
}

func (this *Scheduler) HandleBackendLiveChange(target core.Target, live bool, rtt time.Duration, loss float64) {
	backend, ok := this.backends[target]
	if !ok {
		logging.For("scheduler").Warn("No backends from checkResult ", target)
		return
	}
	backend.Stats.Live = live
	backend.Stats.Rtt = rtt
	backend.Stats.Loss = loss
}

