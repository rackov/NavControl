package sendegts

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/naoina/toml"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/proto"
	log "github.com/sirupsen/logrus"
)

type SendServer struct {
	ServiceName string
	PortList    int
	IpDb        string
	PortDb      int
	User        string
	Passw       string
	DbName      string
	servOut     ArrConn
	db          *sql.DB
	sigChan     chan struct{}
	errorCon    error
	mutex       *sync.Mutex
	restart     bool
	logger      *logger.Logger
}

func Open(fname string) (srv SendServer, er error) {
	srv = SendServer{}

	f, err := os.Open(fname)

	if err != nil {
		return srv, err
	}
	defer f.Close()

	if err = toml.NewDecoder(f).Decode(&srv); err != nil {
		return srv, err
	}

	srv, err = New(&srv)

	return srv, err

}
func (s *SendServer) Save(fname string) error {
	f, err := os.Create(fname)

	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(s)
}
func New(send *SendServer) (sn SendServer, err error) {

	sn = SendServer{
		ServiceName: send.ServiceName,
		PortList:    send.PortList,
		User:        send.User,
		IpDb:        send.IpDb,
		DbName:      send.DbName,
		Passw:       send.Passw,
		PortDb:      send.PortDb,
		servOut:     make(ArrConn, 0),
		mutex:       &sync.Mutex{},
		restart:     false,
	}

	return sn, err
}

func (s *SendServer) sendPkg(db *sql.DB) {
	for _, cl := range s.servOut {
		cl.db = db
		if !cl.Active {
			continue
		}

		go cl.Run()
	}
}
func (s *SendServer) Run() {
	// Запуск -- инициализация
	err := s.db_open()
	if err != nil {
		log.Info("Ошибка подключения к базе ", err)
		return
	}

	err = s.read()
	if err != nil {
		log.Info("Ошибка инициализации списка ", err)

		return
	}
	s.sigChan = make(chan struct{})

	s.sendPkg(s.db)

	for {
		select {
		case <-s.sigChan:
			log.Info("Shutting down ")
			if s.db != nil {
				s.db.Close()
				s.db = nil
			}
			return
		case <-time.After(1000 * time.Millisecond):

			// Если соединение не установлено или оно разорвано, пытаемся переподключиться
			if s.errorCon = s.db.Ping(); s.errorCon != nil {

				errb := s.db_open()
				log.Info("Open db ", errb)
			} else if s.restart {
				log.Info("Restart send ")

				s.restart = false
				s.sendPkg(s.db)

			}

		}
	}

}
func (s *SendServer) StopClient(st *proto.SetClient) (*proto.Client, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, cl := range s.servOut {
		if cl.IdClient == int(st.IdRetranslator) {

			if cl.Active {
				ret, err := s.updateActive(int(st.IdRetranslator), false)
				if err != nil {
					return ret, err
				}

				ret.IdSm = st.IdSm
				cl.Stop()
				cl.Active = false

				return ret, err
			}

			return nil, fmt.Errorf("client is stopped")
		}
	}

	return nil, fmt.Errorf("Not fiend client")
}
func (s *SendServer) RunClient(st *proto.SetClient) (*proto.Client, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, cl := range s.servOut {
		if cl.IdClient == int(st.IdRetranslator) {
			if !cl.Active {
				cl.db = s.db
				ret, err := s.updateActive(int(st.IdRetranslator), true)
				if err == nil {
					ret.IdSm = st.IdSm
					cl.Active = true
					go cl.Run()
					return ret, err
				}

			}
			return nil, fmt.Errorf("client is active")
		}
	}
	return nil, fmt.Errorf("Not fiend client")
}
func (s *SendServer) Stop() error {

	s.mutex.Lock()
	for _, cl := range s.servOut {
		if cl.Active {
			cl.Stop()
		}
	}
	s.mutex.Unlock()
	if s.sigChan != nil {
		close(s.sigChan)
	}

	return nil
}

func (s *SendServer) db_open() error {
	constr := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		s.IpDb, s.PortDb,
		s.User, s.Passw, s.DbName)
	// log.Info("Con DB: ", constr)
	var err error
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
	// Открываем соединение с базой данных

	s.db, err = sql.Open("postgres", constr)
	if err != nil {
		//		log.Info("Ошибка открытия соединения:", err)
		return err
	}
	s.restart = true
	// Проверяем соединение
	if err := s.db.Ping(); err != nil {
		//		log.Info("Ошибка проверки соединения:", err)
		return err
	}
	s.restart = false
	return err

}
func (s *SendServer) ListClient() (*proto.Clients, error) {
	sql := "select ret_json from telematic.list_cl(-1)"
	clients := proto.Clients{}
	rows, err := s.db.Query(sql)

	if err != nil {
		log.Info("Ошибка выполнения запроса:", err)
		return &clients, err
	}

	clients.Clients = make([]*proto.Client, 0)
	for rows.Next() {
		cl := proto.Client{}
		str := ""
		err := rows.Scan(&str)
		if err != nil {
			log.Info("Ошибка чтения строки:", err)
			return &clients, err
		}
		err1 := json.Unmarshal([]byte(str), &cl)
		if err1 != nil {
			log.Info("Ошибка чтения listing client:", err)
			return &clients, err
		}
		if cl.DeviceList == nil {
			cl.DeviceList = make([]int32, 0)
		}

		clients.Clients = append(clients.Clients, &cl)
	}
	if err := rows.Err(); err != nil {
		log.Info("Ошибка перебора строк:", err)
		return &clients, err
	}
	rows.Close()
	return &clients, err
}

// func (s *SendServer) RetLog(arg *proto.ComArgs) (*proto.StateServ, error) {

// 	arg.Args = append(arg.Args, "-u")
// 	arg.Args = append(arg.Args, s.ServiceName)
// 	// log.Info("serv name: ", s.ServiceName)
// 	cmd := exec.Command("journalctl", arg.Args...)
// 	var out bytes.Buffer
// 	cmd.Stdout = &out
// 	var stderr bytes.Buffer
// 	cmd.Stderr = &stderr
// 	// log.Info("Run journalctl", arg.Args)
// 	err := cmd.Run()
// 	if err != nil {
// 		return &proto.StateServ{Message: "Ошибка выполнения "}, err
// 	}

// 	ret := proto.StateServ{Message: out.String()}

//		return &ret, nil
//	}
func (s *SendServer) UpdateClient(upcl *proto.Client) (*proto.Client, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var err error

	for _, cl := range s.servOut {
		if cl.IdClient == int(upcl.IdRetranslator) {

			cl.Param.Ip = upcl.IpClient
			cl.Param.Port = int(upcl.PortClient)
			cl.Param.Protocol = upcl.Protocol
			cl.Field = upcl.Sensors
			upcl.IsActive = cl.Active
			err = s.update(upcl)
			return upcl, err
		}
	}

	return upcl, fmt.Errorf("client not find")
}
func (s *SendServer) ListDevices(set *proto.SetClient) (*proto.Devices, error) {
	var devs proto.Devices
	devs.ListDevice = make([]*proto.Device, 0)

	sql := "SELECT ret_json from telematic.list_device()"
	rows, err := s.db.Query(sql)
	str := ""
	for rows.Next() {
		cl := proto.Device{}
		err = rows.Scan(&str)
		if err != nil {
			log.Info("Ошибка чтения строки:", err)
			return &devs, err
		}
		err1 := json.Unmarshal([]byte(str), &cl)
		if err1 != nil {
			log.Info("Ошибка чтения list_device:", err)
			return &devs, err
		}

		devs.ListDevice = append(devs.ListDevice, &cl)

	}

	return &devs, err
}
func (s *SendServer) DeleteClient(set *proto.SetClient) (oldCl *proto.Client, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	f := -1
	for i, cl := range s.servOut {
		if cl.IdClient == int(set.IdRetranslator) {
			f = i
		}
	}
	if f == -1 {
		return nil, fmt.Errorf("client not find")
	}
	oldCl, err = s.delete(set)
	if err != nil {
		return nil, err
	}
	if s.servOut[f].Active {
		s.servOut[f].Stop()
	}

	oldCl.IdSm = set.IdSm
	copy(s.servOut[f:], s.servOut[f+1:])
	s.servOut[len(s.servOut)-1] = nil
	s.servOut = s.servOut[:len(s.servOut)-1]
	return oldCl, err
}
func (s *SendServer) delete(set *proto.SetClient) (*proto.Client, error) {
	cl := proto.Client{}
	str := ""
	err := s.db.QueryRow(
		`select ret_json from telematic.delete_cl($1)`,
		set.IdRetranslator).Scan(&str)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(str), &cl)
	if err != nil {
		return nil, err
	}

	return &cl, err
}
func (s *SendServer) AddClient(insCl *proto.Client) (newCl *proto.Client, err error) {

	client := SendConnect{
		Param:  SendParam{Ip: insCl.IpClient, Port: int(insCl.PortClient), Protocol: insCl.Protocol},
		Active: insCl.IsActive,
	}
	client.Field = make([]string, 0)
	for _, s := range insCl.Sensors {
		client.Field = append(client.Field, s)
	}
	id, err := s.insertCl(insCl)
	if err != nil {
		log.Info("Error insert DB ", err)
		return insCl, err
	}
	insCl.IdRetranslator = int32(id)
	// fmt.Printf("id %d  idR %d", id, insCl.IdRetranslator)
	client.IdClient = id
	client.db = s.db
	if client.Active {
		go client.Run()
	}

	s.servOut = append(s.servOut, &client)
	return insCl, err
}
func (s *SendServer) updateActive(id_client int, active bool) (*proto.Client, error) {
	cl := proto.Client{}
	str := ""
	err := s.db.QueryRow(
		`select ret_json from telematic.change_active($1,$2)`,
		id_client, active).Scan(&str)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(str), &cl)
	if err != nil {
		return nil, err
	}

	return &cl, err

}
func (s *SendServer) update(upcl *proto.Client) (err error) {
	if upcl.Sensors == nil {
		upcl.Sensors = []string{}
	}
	if upcl.DeviceList == nil {
		upcl.DeviceList = []int32{}
	}

	sp, err := json.Marshal(upcl)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`select * from telematic.update_cl($1)`,
		string(sp))

	return err

}
func (s *SendServer) insertCl(insCl *proto.Client) (id int, err error) {

	if insCl.Sensors == nil {
		insCl.Sensors = make([]string, 0)
	}
	if insCl.DeviceList == nil {
		insCl.DeviceList = make([]int32, 0)
	}

	str, err := json.Marshal(insCl)
	if err != nil {
		return 0, err
	}

	err = s.db.QueryRow(`select insert_cl from telematic.insert_cl($1)`, string(str)).Scan(&id)

	return id, err

}

func (s *SendServer) read() error {
	var list_field string

	sql := `
	SELECT id, ip, port, protocol, list_field, s_active
	FROM telematic.v_list_client; `

	rows, err := s.db.Query(sql)

	if err != nil {
		log.Info("Ошибка выполнения запроса:", err)
		return err
	}
	for rows.Next() {
		cl := SendConnect{}
		err := rows.Scan(&cl.IdClient, &cl.Param.Ip, &cl.Param.Port, &cl.Param.Protocol, &list_field, &cl.Active)
		if err != nil {
			log.Info("Ошибка чтения строки:", err)
			return err
		}
		err1 := json.Unmarshal([]byte(list_field), &cl.Field)
		if err1 != nil {
			log.Info("Ошибка чтения list_field:", err)
			continue
		}
		cl.IdRec = 1
		cl.mutex = &sync.Mutex{}
		s.servOut = append(s.servOut, &cl)
	}
	// Проверяем, есть ли ошибки во время перебора строк
	if err := rows.Err(); err != nil {
		log.Info("Ошибка перебора строк:", err)
		return err
	}
	log.Info("Инициализация клиентов  telematic.v_list_client")
	rows.Close()
	return err

}
