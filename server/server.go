package server

import (
	"net"
	"time"
	"strings"
	"errors"

	"../logging"
	"../config"
	"../scheduler"
	"../discovery"
	"../healthcheck"
	"../balance"
	"../core"
	"../utils/consistent"
	"golang.org/x/net/ipv4"
)

const UDP_PACKET_SIZE = 1500

type Server struct {
	name string
	cfg config.Server

	scheduler *scheduler.Scheduler
	consistent *consistent.Consistent
	serverConn *net.UDPConn
	stopped bool

	liveBackendsMap map[string]*core.Backend
	liveBackends []core.Backend

	getOrCreateChan chan *sessionRequest
	removeChan chan string
	stopChan chan bool
}

type sessionRequest struct {
	clientAddr	net.UDPAddr
	ipv4Header	*ipv4.Header
	response	chan sessionResponse
}

type sessionResponse struct {
	session	*session
	err	error
}

func New(name string, cfg config.Server) (*Server, error) {
	log := logging.For("server")

	consistent := consistent.New()

	scheduler := &scheduler.Scheduler{
		Balancer: balance.New(cfg.Balance),
		Discovery: discovery.New(cfg.Discovery.Kind, *cfg.Discovery),
		Healthcheck: healthcheck.New(cfg.Healthcheck.Kind, *cfg.Healthcheck),
	}
	server := &Server{
		name:			name,
		cfg:			cfg,
		consistent:		consistent,
		scheduler:		scheduler,
		getOrCreateChan:	make(chan *sessionRequest),
		removeChan:		make(chan string),
		stopChan:		make(chan bool),
	}

	log.Info("Creating server '", name, "': ", cfg.Bind);

	return server, nil
}

func (this *Server) Cfg() config.Server {
	return this.cfg
}

func (this *Server) Start() error {
	log := logging.For("Server")

	this.scheduler.Start()

	if err := this.Listen(); err != nil {
		this.Stop()
		log.Error("Error starting listen ", err);
		return err
	}

	go func() {
		sessions := make(map[string]*session)
		for {
			select {
			case sessionRequest := <-this.getOrCreateChan:
				skey, err := this.getSessionKey(sessionRequest)
				if nil != err {
					sessionRequest.response <- sessionResponse{
						session:	nil,
						err:		err,
					}
					break
				}
				log.Debug("getting session: ", skey)
				session, ok :=sessions[skey]
				if ok {
					sessionRequest.response <- sessionResponse{
						session:	session,
						err:		nil,
					}
					break
				}
				session, err = this.makeSession(sessionRequest.clientAddr, skey)
				if err == nil {
					log.Info("new seesion: ", session)
					sessions[skey] = session
				}
				sessionRequest.response <- sessionResponse{
					session:	session,
					err:		err,
				}

			case skey := <-this.removeChan:
				session, ok :=sessions[skey]
				if !ok {
					break
				}
				session.Stop()
				delete(sessions, skey)
			case backends := <-this.scheduler.LiveBackendsChan:
				updated := map[string]*core.Backend{}
				servers := make([]string, len(backends))
				for i := range backends {
					b := backends[i]
					updated[b.Target.String()] = &b
					servers[i] = b.Target.String()
				}
				this.liveBackendsMap = updated
				this.liveBackends = backends
				this.consistent.Set(servers)
				log.Info("live backends:", servers)
				for k, v := range sessions {
					log.Info("session: ", k, "->", v.Backend().Target)
				}
			case <-this.stopChan:
				for _, session := range sessions {
					session.Stop();
				}
				return
			}
		}
	}()

	return nil
}

func (this *Server) Listen() error {
	log := logging.For("server")

	listenAddr, err := net.ResolveUDPAddr("udp", this.cfg.Bind)
	if err != nil {
		log.Error("Error ResolveUDPAddr ", err)
		return err
	}

	this.serverConn, err = net.ListenUDP("udp", listenAddr)
	if err != nil {
		log.Error("Error start server  ", err)
		return err
	}

	go func() {
		for {
			buf := make([]byte, UDP_PACKET_SIZE)
			n, clientAddr, err := this.serverConn.ReadFromUDP(buf)
			if err != nil {
				if this.stopped {
					return
				}
				log.Error("Error ReadFromUDP: ", err)
				continue
			}

			go func(buf []byte) {
				header, _ := ipv4.ParseHeader(buf)
				responseChan := make(chan sessionResponse, 1)
				//log.Debug("session request from ", clientAddr.String(), " header: ", header)
				this.getOrCreateChan <- &sessionRequest{
					clientAddr: *clientAddr,
					ipv4Header: header,
					response: responseChan,
				}

				response := <-responseChan
				if response.err != nil {
					log.Error("Error creating session ", response.err)
					return
				}


				err := response.session.send(buf)
				if err != nil {
					log.Error("Error sending data to backend ", err)
				}

			}(buf[0:n])

		}
	}()

	return nil
}

func (this *Server) getSessionKey(req *sessionRequest) (string, error) {
	log := logging.For("server")

	server, err := this.consistent.Get(req.ipv4Header.Dst.String())

	if nil != err {
		return "", err
	}

	//bucket := hasher.HashString(req.ipv4Header.Dst.String(), int32(len(this.liveBackends)))
	//backend := this.liveBackends[bucket]

	/*

	for _, v := range targets {
		c.Add(v.String())
		log.Debug("addding server :", v)
	}

	log.Debug("hash key: ", req.ipv4Header.Dst)
	server, _ := c.Get(req.ipv4Header.Dst.String())
	*/

	log.Debug("hash server: ", server, " for: ", req.clientAddr, "->", req.ipv4Header.Dst)


	return req.clientAddr.String() + ":" + server, nil
}
func (this *Server) getBackendBySessionKey(sessionKey string) (*core.Backend, error) {
	items := strings.SplitN(sessionKey, ":", 3)
	server := items[len(items)-1]
	backend, ok := this.liveBackendsMap[server]
	if !ok {
		return nil, errors.New("Not found")
	}
	return backend, nil
}

func (this *Server) makeSession(clientAddr net.UDPAddr, sessionKey string) (*session, error) {
	log := logging.For("server")

	/*
	backend, err := this.scheduler.TakeBackend(&core.UdpContext{
		RemoteAddr: clientAddr,
	})
	*/
	backend, err := this.getBackendBySessionKey(sessionKey)
	if err != nil {
		log.Error("Error take backend server", err)
		return nil, err
	}

	backendTimeout, err := time.ParseDuration("0s")

	session := &session{
		backendIdleTimeout: backendTimeout,
		serverConn: this.serverConn,
		clientAddr: clientAddr,
		sessionKey: sessionKey,
		notifyClosed: func() {
			this.removeChan <- sessionKey
		},
		backend: backend,
	}

	err = session.Start()
	if err != nil {
		session.Stop()
		return nil, err
	}

	return session, nil

}

func (this *Server) Stop() {
	log := logging.For("server")
	log.Info("Stopping ", this.name)

	this.stopped = true
	this.serverConn.Close()

	this.scheduler.Stop()
	this.stopChan <- true
}
