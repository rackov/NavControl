// go run *.go -config "/home/vladimir/go/project/NavControl/cfg/writer.toml"
package wrdbnats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"database/sql"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
)

const (
	StConOK   = 0
	StConStop = 50
	StConBase = 100
	StNotCon  = 200
	maxlen    = 900
)

type StatusConn struct {
	Id    int
	Error error
}

func isValidJSON(data []byte) error {
	var jsonMap map[string]interface{} // Используем map, чтобы обработать любой JSON
	err := json.Unmarshal((data), &jsonMap)

	return err
}

type WrNatsServer struct {
	IdSm         int
	IdWriter     int
	tempFileName string
	stcon        StatusConn
	con_str      string
	ttN          time.Duration
	sqldb        *sql.DB
	nsAdrr       string //Nats
	natsKey      string
	nc           *nats.Conn
	sigChan      chan struct{}
	Active       bool
	tempFile     *os.File
	mutex        *sync.Mutex
	writeData    chan string
	log          *logger.Logger
}

func (s *WrNatsServer) init() (err error) {
	// Создание временного файла
	if _, err := os.Stat(s.tempFileName); err == nil {
		// Если файл существует
		s.tempFile, _ = os.OpenFile(s.tempFileName, os.O_WRONLY, 0666)
	} else if os.IsNotExist(err) {
		// Если файл не существует, создаем его
		s.tempFile, s.stcon.Error = os.Create(s.tempFileName)
		if s.stcon.Error != nil {
			s.log.Info("Error creating temporary file: ", s.stcon.Error)
			return s.stcon.Error
		}
	}

	// Устанавливаем соединение с NATS
	s.nc, s.stcon.Error = nats.Connect(s.nsAdrr)
	if s.stcon.Error != nil {
		s.log.Info("Error connecting to NATS: ", s.stcon.Error)
		return s.stcon.Error
	}

	// Устанавливаем соединение с PostgreSQL
	s.sqldb, s.stcon.Error = sql.Open("postgres", s.con_str)

	s.stcon.Error = s.sqldb.Ping()
	if s.stcon.Error != nil {
		s.log.Info("Unable to connect to database: ", s.stcon.Error)
		return s.stcon.Error
	}
	s.stcon.Id = StConOK
	return s.stcon.Error
}

func New(wr *config.Writer, NatsAddress string, NatsTopic string, log *logger.Logger) (wrN WrNatsServer, err error) {

	tempfile := fmt.Sprintf("temp_data_%d.json", wr.IdWriter)
	constr := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		wr.DbIp, wr.DbPort,
		wr.DbUser, wr.DbPass, wr.DbName)

	wrN = WrNatsServer{
		tempFileName: tempfile,
		stcon:        StatusConn{Id: StNotCon, Error: nil},
		con_str:      constr,
		nsAdrr:       NatsAddress,
		natsKey:      NatsTopic,
		Active:       true,
		ttN:          5 * time.Second,
		log:          log,
		mutex:        &sync.Mutex{},
	}
	wrN.stcon.Error = wrN.init()
	return wrN, wrN.stcon.Error
}

func (s *WrNatsServer) Run() {

	s.log.Info("Run: ", s.natsKey)
	if s.tempFileName == "" {
		return
	} else {
		s.init()
	}

	// Получаем данные из NATS
	_, s.stcon.Error = s.nc.Subscribe(s.natsKey, s.processMessage)
	if s.stcon.Error != nil {
		s.log.Info("Error subscribing to NATS: ", s.stcon.Error,
			" sub:", s.natsKey)
	}

	// Запуск горутины для выгрузки данных из временного файла в базу
	s.Active = true
	go s.processTempFile()

	defer s.tempFile.Close()
	defer s.sqldb.Close()
	buf := make([]string, maxlen)
	wi := 0
	r := 0
	s.sigChan = make(chan struct{})
	s.writeData = make(chan string)

	for {
		select {
		case <-s.sigChan:
			s.log.Infof("Shutting down :%s", s.natsKey)
			s.Active = false
			return
		case newValue := <-s.writeData:
			value := newValue
			wi = wi + 1

			if r == (maxlen - 1) {
				buf[r] = value
				s.saveToDatabase(buf, r)
				r = 0
			} else {
				buf[r] = value
				r = r + 1

			}
			// log.Info("Val", value)
		case <-time.After(500 * time.Millisecond):
			if r > 0 {
				s.saveToDatabase(buf, r)
				r = 0
			}
		case <-time.After(s.ttN):
			// Если соединение не установлено или оно разорвано, пытаемся переподключиться
			if s.stcon.Error = s.sqldb.Ping(); s.stcon.Error != nil {
				s.connect()
			}
			s.log.Info("Проверка соединения")
			// time.Sleep(s.ttN)
		}
	}

}

func (s *WrNatsServer) connect() {
	s.mutex.Lock()

	s.stcon.Error = nil
	defer s.mutex.Unlock()

	s.sqldb, s.stcon.Error = sql.Open("postgres", s.con_str)

	if s.stcon.Error != nil {
		s.stcon.Id = StNotCon
		s.log.Info("Unable to connect to database: ", s.stcon.Error)

		return
	}

	// Проверяем подключение
	if s.stcon.Error = s.sqldb.Ping(); s.stcon.Error != nil {
		s.stcon.Id = StNotCon
		s.log.Info("Unable to reach database: ", s.stcon.Error)
		s.sqldb.Close()
		return
	}
	s.stcon.Id = StConOK

}
func (s *WrNatsServer) Stop() error {

	s.mutex.Lock()
	if s.stcon.Id != StConStop {
		close(s.sigChan)
	}
	if s.nc != nil {
		s.nc.Close()
	}
	s.stcon.Id = StConStop
	s.Active = false
	s.mutex.Unlock()
	return nil
}

// Данные с Nats
func (s *WrNatsServer) processMessage(msg *nats.Msg) {

	s.nc.Publish(msg.Reply, []byte("ok"))

	err := isValidJSON(msg.Data)
	if err != nil {
		//		log.Printf("Error unmarshalling message: %v", err)
		return
	}

	s.writeData <- string(msg.Data)

}

// запись в базу
func (s *WrNatsServer) saveToDatabase(buf []string, l int) {
	var strsql string = "insert into one.temp(fjson) values "
	err := s.sqldb.Ping()
	if err == nil {
		for i := 0; i < l; i++ {
			strsql = strsql + fmt.Sprintf("('%s')", buf[i])

			if i != (l - 1) {
				strsql = strsql + ","
			}

		}
		_, err = s.sqldb.Exec(strsql)
	}
	if err != nil {
		for i := 0; i < l; i++ {
			s.saveToTempFile([]byte(buf[i]))
		}
	}

}

// запись в файл
func (s *WrNatsServer) saveToTempFile(data []byte) error {
	s.mutex.Lock()

	if _, err := s.tempFile.Write(append(data, '\n')); err != nil {
		s.mutex.Unlock()
		return err

	}
	s.tempFile.Sync()
	s.mutex.Unlock()
	return nil
}

func (s *WrNatsServer) processTempFile() {
	t := 0

	for {
	TOD:
		time.Sleep(500 * time.Millisecond) // Пауза между проверками
		s.mutex.Lock()
		if !s.Active {
			s.mutex.Unlock()
			return
		}
		if s.stcon.Id == StNotCon {
			s.mutex.Unlock()
			continue
		}
		s.mutex.Unlock()
		if t < 10 {
			t = t + 1
			continue
		} else {
			t = 0
		}
		err := s.sqldb.Ping()
		if err != nil {
			goto TOD
		}
		data, err := os.ReadFile(s.tempFileName)
		if err != nil {
			s.log.Infof("Error reading temporary file: %v", err)
			continue
		}

		if len(data) == 0 {
			continue // Файл пуст, пропустить итерацию
		}

		// Здесь мы можем парсить данные и записывать их в базу
		st_t := time.Now()
		s.log.Infof("Start %s", st_t)
		strsql := ""
		i := 0
		tempData := bytes.Split(data, []byte{'\n'})
		for _, line := range tempData {
			if len(line) == 0 {
				continue // Пропускаем пустые строки
			}
			// Пытаемся записать в базу
			// if err = s.saveToDatabase(line); err != nil {
			// 	log.Printf("Error saving temp data to database: %v", err)
			// }

			if i < maxlen {
				i = i + 1
				strsql = strsql + "('" + string(line) + "'),"
			} else {
				i = 0
				strsql = "insert into one.temp(fjson) values" +
					strsql + "('" + string(line) + "')"
				_, err = s.sqldb.Exec(strsql)

				if err != nil {
					s.log.Printf("Error saving for temp data to database: %v", err)
					strsql = ""
					// s.tempFile.Seek(0, 0)
					goto TOD
				}
				strsql = ""
			}

		}
		if len(strsql) != 0 {
			strsql = strsql[:(len(strsql) - 1)]
			strsql = "insert into one.temp(fjson) values" + strsql
			_, err = s.sqldb.Exec(strsql)
			if err != nil {
				s.log.Printf("Error saving last temp to database: %v", err)
				// log.Printf("%s", strsql)
			}
		}
		s.log.Infof("Duration %f", time.Since(st_t).Seconds())
		// Очищаем временный файл после успешной обработки
		s.mutex.Lock()
		if err == nil {
			s.tempFile.Truncate(0)
		}
		s.tempFile.Seek(0, 0)
		s.mutex.Unlock()
	}
}
