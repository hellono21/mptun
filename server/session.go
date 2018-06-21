package server

import (
	"net"
	"time"

	"../logging"
	"../core"
)

type session struct {
	serverConn *net.UDPConn
	clientAddr net.UDPAddr
	backend *core.Backend
	backendIdleTimeout time.Duration
	backendConn *net.UDPConn
	sessionKey string

	stopC chan bool
	notifyClosed func()
}

func (s *session) Start() error {
	log := logging.For("server/session")
	s.stopC = make(chan bool)

	backendAddr, err := net.ResolveUDPAddr("udp", s.backend.Target.String())

	if err != nil {
		log.Error("Error ResolveUDPAddr: ", err)
		return err
	}

	backendConn, err := net.DialUDP("udp", nil, backendAddr)

	if err != nil {
		log.Debug("Error connecting to backend: ", err)
		return err
	}

	s.backendConn = backendConn

	stopped := false

	go func() {
		for {
			select {
			case <-s.stopC:
				stopped = true
				log.Info("Closing client session: ", s.sessionKey )
				s.backendConn.Close()
				s.notifyClosed()
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, UDP_PACKET_SIZE)

		for {
			if s.backendIdleTimeout > 0 {
				err := s.backendConn.SetReadDeadline(time.Now().Add(s.backendIdleTimeout))
				if err != nil {
					log.Error("Unable to set timeout for backend. closing . Error :", err)
					s.Stop()
					return
				}
			}
			n, _, err := s.backendConn.ReadFromUDP(buf)
			if err != nil {
				if !err.(*net.OpError).Timeout() && !stopped {
					log.Error("Error reading from backend ", err)
				}
				s.Stop()
				return
			}
			s.serverConn.WriteToUDP(buf[0:n], &s.clientAddr)
		}
	}()

	return nil
}

func (s *session) Backend() *core.Backend {
	return s.backend
}


func (s *session) send(buf []byte) error {
	_, err := s.backendConn.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func (c *session) Stop() {
	select {
	case c.stopC <- true:
	default:
	}
}
