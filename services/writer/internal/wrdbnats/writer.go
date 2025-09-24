package wrdbnats

import (
	"encoding/json"

	"sync"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
)

type HubServer struct {
	hubClients []*WrNatsServer
	register   chan *WrNatsServer
	unregister chan *WrNatsServer
	config     *config.ControlWriter
	logger     *logger.Logger
	mutex      *sync.Mutex
}

func NewHubServer(config *config.ControlWriter, log *logger.Logger) *HubServer {
	return &HubServer{
		config:     config,
		logger:     log,
		hubClients: make([]*WrNatsServer, 0),
		register:   make(chan *WrNatsServer),
		unregister: make(chan *WrNatsServer),
		mutex:      &sync.Mutex{},
	}
}
func (hs *HubServer) GetConfig() *config.ControlWriter {
	return hs.config
}
func (hs *HubServer) GetLogger() *logger.Logger {
	return hs.logger
}
func (hs *HubServer) ListService() string {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()
	s, _ := json.Marshal(&hs.hubClients)
	return string(s)
}

func (hs *HubServer) Run() {
	for {
		select {
		case hclient := <-hs.register:
			hs.regServis(hclient)
		case hclient := <-hs.unregister:
			hs.unregServis(hclient)
		}
	}
}

func (hs *HubServer) regServis(hclient *WrNatsServer) {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()
	hs.hubClients = append(hs.hubClients, hclient)
}
func (hs *HubServer) unregServis(hclient *WrNatsServer) {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()
	i := -1
	for j, c := range hs.hubClients {
		if (c.IdSm == hclient.IdSm) && (c.IdWriter == hclient.IdWriter) {
			i = j
			break
		}
	}
	if i != -1 {
		copy(hs.hubClients[i:], hs.hubClients[i+1:])
		hs.hubClients[len(hs.hubClients)-1] = nil
		hs.hubClients = hs.hubClients[:len(hs.hubClients)-1]
	}
}
func (hs *HubServer) UppService(idSm int, idWriter int) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()
	var (
		err error
	)
	for _, c := range hs.hubClients {
		if (c.IdSm == idSm) && (c.IdWriter == idWriter) {
			c.Run()
			break
		}
	}
	return err
}

func (hs *HubServer) DownService(idSm int, idWriter int) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()
	var (
		err error
	)
	for _, c := range hs.hubClients {
		if (c.IdSm == idSm) && (c.IdWriter == idWriter) {
			err = c.Stop()
			break
		}
	}

	return err
}

func (hs *HubServer) AddService() error {
	hclient, err := New(hs.config)
	if err != nil {
		// log.Info("Err Write ", err)
		return err
	}
	go hclient.Run()

	hs.register <- &hclient
	return err
}

func (hs *HubServer) DeleteService(idSm int, idWriter int) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()
	var (
		err error
	)
	for _, c := range hs.hubClients {
		if (c.IdSm == idSm) && (c.IdWriter == idWriter) {
			err = c.Stop()
			hs.unregister <- c
			break
		}
	}
	return err
}
