/**
 * worker.go - Healtheck worker
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package healthcheck

import (
	"../config"
	"../core"
	"../logging"
	"time"
)

/**
 * Healthcheck Worker
 * Handles all periodic healthcheck logic
 * and yields results on change
 */
type Worker struct {

	/* Target to monitor and check */
	target core.Target

	/* Function that does actual check */
	check CheckFunc

	/* Channel to write changed check results */
	out chan<- CheckResult

	/* Healthcheck configuration */
	cfg config.HealthcheckConfig

	/* Stop channel to worker to stop */
	stop chan bool

	/* Current passes count, if LastResult.Live = true */
	passes int

	/* Current fails count, if LastResult.Live = false */
	fails int

	rtt time.Duration
}

/**
 * Start worker
 */
func (this *Worker) Start() {

	log := logging.For("healthcheck/worker")

	// Special case for no healthcheck, don't actually start worker
	if this.cfg.Kind == "none" {
		return
	}

	interval, _ := time.ParseDuration(this.cfg.Interval)

	ticker := time.NewTicker(interval)
	c := make(chan CheckResult, 1)

	go func() {
		for {
			select {

			/* new check interval has reached */
			case <-ticker.C:
				log.Debug("Next check ", this.cfg.Kind, " for ", this.target)
				go this.check(this.target, this.cfg, c)

			/* new check result is ready */
			case checkResult := <-c:
				log.Debug("Got check result ", this.cfg.Kind, ": ", checkResult)
				this.process(checkResult)

			/* request to stop worker */
			case <-this.stop:
				ticker.Stop()
				//close(c) // TODO: Check!
				return
			}
		}
	}()
}

/**
 * Process next check result,
 * counting passes and fails as needed, and
 * sending updated check result to out
 */
func (this *Worker) process(checkResult CheckResult) {

	log := logging.For("healthcheck/worker")

	if !checkResult.Live {
		this.fails ++
	} else {
		this.passes ++
		this.rtt += checkResult.Rtt
	}

	if (this.passes + this.fails) >= this.cfg.Count {
		loss := float64(this.fails)/float64(this.passes + this.fails)
		rtt := 0.0
		if this.passes > 0 {
			rtt = float64(this.rtt)/float64(this.passes)
		}
		cfgRtt, _ := time.ParseDuration(this.cfg.Rtt)
		checkResult.Rtt = time.Duration(rtt)
		checkResult.Loss = loss
		if loss >= this.cfg.Loss || time.Duration(rtt) > cfgRtt {
			checkResult.Live = false
		} else {
			checkResult.Live = true
		}

		log.Info("Sending to scheduler: ", checkResult)
		this.out <- checkResult
		this.reset()
	}
}

func (this *Worker) reset() {
	this.rtt = time.Duration(0)
	this.passes = 0
	this.fails = 0
}

/**
 * Stop worker
 */
func (this *Worker) Stop() {
	close(this.stop)
}
