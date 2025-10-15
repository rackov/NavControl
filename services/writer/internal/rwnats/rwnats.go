package rwnats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rackov/NavControl/pkg/logger"
)

type WrNatsServer struct {
	IsJetStream bool
	IdWriter    int32
	IdSm        int32
	IpDb        string
	PortDb      int32
	DbName      string
	DbTable     string
	Login       string
	Passw       string
	Name        string
	Description string
	IpBroker    string
	PortBroker  int32
	TopicBroker string
	Log         *logger.Logger
	SizeBuf     int

	// Дополнительные поля для внутреннего использования
	db            *pgxpool.Pool
	nc            *nats.Conn
	js            nats.JetStreamContext
	sub           *nats.Subscription
	tempFile      *os.File
	dbConnected   bool
	natsConnected bool

	// Buffer for batch inserts
	dataBuffer []string
	bufferMu   sync.Mutex
}

// // New создает новый экземпляр WrNatsServer
// func New(idWriter, idSm int32, ipDb string, portDb int32, dbName, dbTable, login, passw, name, description, ipBroker string, portBroker int32, topicBroker string, log *logger.Logger, sizeBuf int, isJetStream bool) *WrNatsServer {
// 	return &WrNatsServer{
// 		IsJetStream: isJetStream,
// 		IdWriter:    idWriter,
// 		IdSm:        idSm,
// 		IpDb:        ipDb,
// 		PortDb:      portDb,
// 		DbName:      dbName,
// 		DbTable:     dbTable,
// 		Login:       login,
// 		Passw:       passw,
// 		Name:        name,
// 		Description: description,
// 		IpBroker:    ipBroker,
// 		PortBroker:  portBroker,
// 		TopicBroker: topicBroker,
// 		Log:         log,
// 		SizeBuf:     sizeBuf,
// 	}
// }

// Start запускает сервер NATS writer
func (w *WrNatsServer) Start() error {
	// Инициализация подключений
	err := w.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Запуск подписки
	err = w.StartSubscription()
	if err != nil {
		return fmt.Errorf("failed to start subscription: %w", err)
	}

	// Запуск мониторинга соединений в отдельной горутине
	go w.MonitorConnections()

	return nil
}

func isValidJSON(data []byte) error {
	var jsonMap map[string]interface{} // Используем map, чтобы обработать любой JSON
	err := json.Unmarshal((data), &jsonMap)

	return err
}
func (w *WrNatsServer) DbConnected() bool {
	return w.dbConnected
}
func (w *WrNatsServer) NatsConnected() bool {
	return w.natsConnected
}

// Init инициализирует подключения к NATS и PostgreSQL
func (w *WrNatsServer) Init() error {
	// Initialize buffer
	w.dataBuffer = make([]string, 0, w.SizeBuf)

	w.Log.Infof("Connecting to NATS : %s", w.IpBroker)
	// Подключение к NATS
	err := w.connectToNATS()
	if err != nil {
		w.Log.Errorf("Failed to connect to NATS: %v", err)
	}

	// Подключение к PostgreSQL
	err = w.connectToDB()
	if err != nil {
		w.Log.Errorf("Failed to connect to DB: %v", err)
		// Создаем временный файл для хранения данных
		w.createTempFile()
	}

	return nil
}

// connectToNATS устанавливает соединение с NATS
func (w *WrNatsServer) connectToNATS() error {
	var err error

	url := fmt.Sprintf("nats://%s:%d", w.IpBroker, w.PortBroker)
	w.nc, err = nats.Connect(url)
	if err != nil {
		w.natsConnected = false
		return err
	}

	if w.IsJetStream {
		w.js, err = w.nc.JetStream()
		if err != nil {
			w.natsConnected = false
			return err
		}
	}

	w.natsConnected = true
	w.Log.Infof("Connected to NATS at %s", url)
	return nil
}

// connectToDB устанавливает соединение с PostgreSQL
func (w *WrNatsServer) connectToDB() error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		w.Login, w.Passw, w.IpDb, w.PortDb, w.DbName)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		w.dbConnected = false
		return err
	}

	w.db, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		w.dbConnected = false
		return err
	}

	// Проверяем соединение
	err = w.db.Ping(context.Background())
	if err != nil {
		w.dbConnected = false
		return err
	}

	w.dbConnected = true
	w.Log.Infof("Connected to PostgreSQL at %s:%d/%s", w.IpDb, w.PortDb, w.DbName)

	// Если база данных снова доступна, записываем данные из временного файла
	if w.tempFile != nil {
		w.flushTempFile()
	}

	return nil
}

// createTempFile создает временный файл для хранения данных при недоступной БД
func (w *WrNatsServer) createTempFile() error {
	filename := fmt.Sprintf("%d_%d.dat", w.IdWriter, w.IdSm)

	var err error
	w.tempFile, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		w.Log.Errorf("Failed to create temp file: %v", err)
		return err
	}

	w.Log.Infof("Created temp file: %s", filename)
	return nil
}

// flushTempFile записывает данные из временного файла в базу данных
func (w *WrNatsServer) flushTempFile() {
	if w.tempFile == nil || !w.dbConnected {
		return
	}

	// Закрываем файл для чтения
	w.tempFile.Close()

	filename := fmt.Sprintf("%d_%d.dat", w.IdWriter, w.IdSm)

	// Открываем файл для чтения
	file, err := os.Open(filename)
	if err != nil {
		w.Log.Errorf("Failed to open temp file for reading: %v", err)
		return
	}
	defer file.Close()

	// Читаем все строки из файла
	data := []string{}
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil && n == 0 {
			break
		}
		data = append(data, strings.Split(string(buf[:n]), "\n")...)
	}

	// Записываем данные в базу
	for _, line := range data {
		if line != "" {
			err := w.writeToDB(line)
			if err != nil {
				w.Log.Errorf("Failed to write data from temp file to DB: %v", err)
				return
			}
		}
	}

	// Удаляем временный файл
	err = os.Remove(filename)
	if err != nil {
		w.Log.Errorf("Failed to remove temp file: %v", err)
	} else {
		w.tempFile = nil
		w.Log.Infof("Flushed temp file and removed it")
	}
}

// StartSubscription запускает подписку на NATS
func (w *WrNatsServer) StartSubscription() error {
	var err error

	if !w.natsConnected {
		return fmt.Errorf("NATS is not connected")
	}

	if w.IsJetStream {
		// Создаем или используем существующий поток JetStream
		streamName := fmt.Sprintf("STREAM_%d_%d", w.IdWriter, w.IdSm)
		_, err = w.js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{w.TopicBroker},
		})
		if err != nil {
			w.Log.Errorf("Failed to add JetStream stream: %v", err)
			return err
		}

		// Подписываемся с помощью JetStream
		w.sub, err = w.js.Subscribe(w.TopicBroker, func(msg *nats.Msg) {
			if err := isValidJSON(msg.Data); err != nil {
				w.Log.Errorf("Error unmarshalling message: %v", err)
				return

			}
			w.handleMessage(string(msg.Data))
			msg.Ack()
		})
	} else {
		// Обычная подписка
		w.sub, err = w.nc.Subscribe(w.TopicBroker, w.processMessage)
	}

	if err != nil {
		w.Log.Errorf("Failed to subscribe to NATS topic: %v", err)
		return err
	}

	w.Log.Infof("Subscribed to NATS topic: %s", w.TopicBroker)
	return nil
}
func (w *WrNatsServer) processMessage(msg *nats.Msg) {

	w.nc.Publish(msg.Reply, []byte("ok"))

	err := isValidJSON(msg.Data)
	if err != nil {
		w.Log.Errorf("Error unmarshalling message: %v", err)
		return
	}

	w.handleMessage(string(msg.Data))

}

// handleMessage обрабатывает сообщение из NATS
func (w *WrNatsServer) handleMessage(data string) {
	// Быстро добавляем данные в буфер без блокировки БД
	w.bufferMu.Lock()
	w.dataBuffer = append(w.dataBuffer, data)
	shouldFlush := len(w.dataBuffer) >= w.SizeBuf
	bufferCopy := make([]string, len(w.dataBuffer))
	copy(bufferCopy, w.dataBuffer)
	if shouldFlush {
		w.dataBuffer = w.dataBuffer[:0] // Очищаем буфер
	}
	w.bufferMu.Unlock()

	// Если буфер полон, записываем данные в БД в отдельной горутине
	if shouldFlush {
		go w.writeBatchToDB(bufferCopy)
	}
}

// writeBatchToDB записывает пакет данных в PostgreSQL
func (w *WrNatsServer) writeBatchToDB(data []string) {
	if !w.dbConnected {
		// Если база данных недоступна, записываем в временный файл
		w.bufferMu.Lock()
		defer w.bufferMu.Unlock()

		if w.tempFile == nil {
			w.createTempFile()
		}

		for _, item := range data {
			w.writeToTempFile(item)
		}
		return
	}

	// Создаем SQL-запрос с множественными значениями
	if len(data) == 0 {
		return
	}

	valueStrings := make([]string, 0, len(data))
	valueArgs := make([]interface{}, 0, len(data))

	for i, item := range data {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d)", i+1))
		valueArgs = append(valueArgs, item)
	}

	query := fmt.Sprintf("INSERT INTO %s VALUES %s", w.DbTable, strings.Join(valueStrings, ","))

	_, err := w.db.Exec(context.Background(), query, valueArgs...)
	if err != nil {
		w.Log.Errorf("Failed to write batch to DB: %v", err)
		// При ошибке записи создаем временный файл
		w.bufferMu.Lock()
		defer w.bufferMu.Unlock()

		if w.tempFile == nil {
			w.createTempFile()
		}

		for _, item := range data {
			w.writeToTempFile(item)
		}
	}
}

// writeToDB записывает одну запись в PostgreSQL (для совместимости и использования в flushTempFile)
func (w *WrNatsServer) writeToDB(data string) error {
	if !w.dbConnected {
		return fmt.Errorf("database is not connected")
	}

	query := fmt.Sprintf("INSERT INTO %s VALUES ($1)", w.DbTable)
	_, err := w.db.Exec(context.Background(), query, data)
	return err
}

// writeToTempFile записывает данные во временный файл
func (w *WrNatsServer) writeToTempFile(data string) {
	if w.tempFile == nil {
		w.Log.Error("Temp file is not initialized")
		return
	}

	_, err := w.tempFile.WriteString(data + "\n")
	if err != nil {
		w.Log.Errorf("Failed to write to temp file: %v", err)
	}
}

// MonitorConnections мониторит подключения к NATS и PostgreSQL
func (w *WrNatsServer) MonitorConnections() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Проверяем подключение к NATS
		if w.nc != nil && w.nc.Status() != nats.CONNECTED {
			w.natsConnected = false
			w.Log.Warn("NATS connection lost. Trying to reconnect...")
			w.connectToNATS()
		}

		// Проверяем подключение к PostgreSQL
		if w.db != nil {
			err := w.db.Ping(context.Background())
			if err != nil {
				if w.dbConnected {
					w.dbConnected = false
					w.Log.Warn("PostgreSQL connection lost. Trying to reconnect...")
				}
				w.connectToDB()
			} else if !w.dbConnected {
				w.dbConnected = true
				w.Log.Info("PostgreSQL connection restored")
				// При восстановлении подключения записываем данные из временного файла
				w.flushTempFile()
			}
		}

		// Проверяем, есть ли необработанные данные в буфере
		w.bufferMu.Lock()
		if len(w.dataBuffer) > 0 && w.dbConnected {
			bufferCopy := make([]string, len(w.dataBuffer))
			copy(bufferCopy, w.dataBuffer)
			w.dataBuffer = w.dataBuffer[:0] // Очищаем буфер
			w.bufferMu.Unlock()

			// Записываем оставшиеся данные в БД
			go w.writeBatchToDB(bufferCopy)
		} else {
			w.bufferMu.Unlock()
		}
	}
}

// Close закрывает все соединения
func (w *WrNatsServer) Close() {
	// Записываем оставшиеся данные перед закрытием
	w.bufferMu.Lock()
	if len(w.dataBuffer) > 0 {
		bufferCopy := make([]string, len(w.dataBuffer))
		copy(bufferCopy, w.dataBuffer)
		w.bufferMu.Unlock()

		if w.dbConnected {
			w.writeBatchToDB(bufferCopy)
		} else if w.tempFile != nil {
			for _, item := range bufferCopy {
				w.writeToTempFile(item)
			}
		}
	} else {
		w.bufferMu.Unlock()
	}

	if w.sub != nil {
		w.sub.Unsubscribe()
	}

	if w.nc != nil {
		w.nc.Close()
	}

	if w.db != nil {
		w.db.Close()
	}

	if w.tempFile != nil {
		w.tempFile.Close()
	}
}
