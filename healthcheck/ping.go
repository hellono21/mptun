/**
 * ping.go - TCP ping healthcheck
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package healthcheck

import (
	"../config"
	"../core"
	"../logging"
	"net"
	"time"
)

/**
 * Ping healthcheck
 */
func ping(t core.Target, cfg config.HealthcheckConfig, result chan<- CheckResult) {

	pingTimeoutDuration, _ := time.ParseDuration(cfg.Timeout)

	log := logging.For("healthcheck/ping")

	checkResult := CheckResult{
		Target: t,
	}

	buf := make([]byte, 20)
	var conn *net.UDPConn
	startT := time.Now()
	tAddr, err := net.ResolveUDPAddr("udp", t.String())
	if nil == err {
		log.Debug("connecting ", tAddr)
		conn, err = net.DialUDP("udp", nil, tAddr);
		defer conn.Close()
	}
	if err != nil {
		log.Debug("DialUDP error ", err)
	}
	if nil == err {
		_, err = conn.Write(buf)
		if err != nil {
			log.Debug("DialUDP error ", err)
		}
		conn.SetReadDeadline(time.Now().Add(pingTimeoutDuration))
		_ , _, err = conn.ReadFromUDP(buf)
	}
	if err != nil {
		log.Debug("Check result false: ", err)
		checkResult.Live = false
	} else {
		checkResult.Live = true
		checkResult.Rtt = time.Now().Sub(startT)
		log.Debug("Rtt: ", checkResult.Rtt.Nanoseconds())
	}

	select {
	case result <- checkResult:
	default:
		log.Warn("Channel is full. Discarding value")
	}
}
