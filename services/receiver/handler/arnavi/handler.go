// pkg/arnavi/arnavi.go
package arnavi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/services/receiver/internal/protocol"
	"github.com/sirupsen/logrus"
)

var (
	connectedDevices = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "navcontrol_connected_devices_total",
			Help: "Total number of currently connected devices",
		},
	)
)

func init() {
	// Регистрация метрики в Prometheus
	prometheus.MustRegister(connectedDevices)
}

// ClientInfo содержит информацию о подключенном клиенте
type ClientInfo struct {
	ID          string
	RemoteAddr  string
	ConnectTime string
	Protocol    string
}

// ArnaviProtocol реализует интерфейс NavigationProtocol для протокола Arnavi
type ArnaviProtocol struct {
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
	clients     map[string]ClientInfo
	clientsMu   sync.Mutex
	connections map[net.Conn]struct{}
	connMu      sync.Mutex
	logger      *logger.Logger
}

// NewArnaviProtocol создает новый экземпляр протокола Arnavi
func NewArnaviProtocol(log *logger.Logger) protocol.NavigationProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	return &ArnaviProtocol{
		ctx:         ctx,
		cancel:      cancel,
		clients:     make(map[string]ClientInfo),
		connections: make(map[net.Conn]struct{}),
		logger:      log,
	}
}

// GetName возвращает имя протокола
func (a *ArnaviProtocol) GetName() string {
	return "Arnavi"
}

// Start запускает TCP-сервер для приема данных
func (a *ArnaviProtocol) Start(port int, nc models.NatsConf) error {
	var err error
	a.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	a.logger.Infof("%s protocol started on port %d", a.GetName(), port)

	// Запускаем обработчик соединений в отдельной горутине
	go a.handleConnections(nc)

	return nil
}

// handleConnections обрабатывает входящие соединения
func (a *ArnaviProtocol) handleConnections(nc models.NatsConf) {
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			conn, err := a.listener.Accept()
			if err != nil {
				// Проверяем, не был ли сервер остановлен
				if a.ctx.Err() != nil {
					return
				}
				a.logger.Errorf("Error accepting connection: %v", err)
				continue
			}

			// Добавляем соединение в список
			a.connMu.Lock()
			a.connections[conn] = struct{}{}
			a.connMu.Unlock()

			// Создаем информацию о клиенте
			clientID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().Unix())
			clientInfo := ClientInfo{
				ID:          clientID,
				RemoteAddr:  conn.RemoteAddr().String(),
				Protocol:    a.GetName(),
				ConnectTime: time.Now().Format(time.RFC3339),
			}

			//	"Client %s connected  Protocol: %s", clientID, a.GetName())
			connectedDevices.Inc()
			// Добавляем клиента в список
			a.clientsMu.Lock()
			a.clients[clientID] = clientInfo
			a.clientsMu.Unlock()

			// Обрабатываем соединение в отдельной горутине
			go a.handleConnection(conn, clientID, nc)
		}
	}
}

// Stop останавливает TCP-сервер
func (a *ArnaviProtocol) Stop() error {
	// Отменяем контекст
	a.cancel()

	// Закрываем все активные соединения
	a.connMu.Lock()
	for conn := range a.connections {
		conn.Close()
	}
	a.connections = make(map[net.Conn]struct{})
	a.connMu.Unlock()

	// Закрываем слушатель
	if a.listener != nil {
		if err := a.listener.Close(); err != nil {
			return fmt.Errorf("error closing listener: %v", err)
		}
	}

	a.logger.Infof("%s protocol stopped", a.GetName())
	return nil
}

// GetClients возвращает список подключенных клиентов
func (a *ArnaviProtocol) GetClients() []ClientInfo {
	a.clientsMu.Lock()
	defer a.clientsMu.Unlock()

	clients := make([]ClientInfo, 0, len(a.clients))
	for _, client := range a.clients {
		clients = append(clients, client)
	}

	return clients
}

// DisconnectClient отключает клиента по ID
func (a *ArnaviProtocol) DisconnectClient(clientID string) error {
	a.clientsMu.Lock()
	client, exists := a.clients[clientID]
	a.clientsMu.Unlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	// Ищем соединение клиента и закрываем его
	a.connMu.Lock()
	defer a.connMu.Unlock()

	for conn := range a.connections {
		if conn.RemoteAddr().String() == client.RemoteAddr {
			conn.Close()
			return nil
		}
	}

	return fmt.Errorf("connection for client %s not found", clientID)
}

// handleConnection обрабатывает одно соединение
func (a *ArnaviProtocol) handleConnection(conn net.Conn, clientID string, nc models.NatsConf) {
	logcl := a.logger.WithFields(
		logrus.Fields{"client": clientID,
			"protocol": a.GetName(),
		})

	defer func() {
		conn.Close()

		// Удаляем соединение из списка
		a.connMu.Lock()
		delete(a.connections, conn)
		a.connMu.Unlock()

		// Удаляем клиента из списка
		a.clientsMu.Lock()
		delete(a.clients, clientID)
		a.clientsMu.Unlock()

		connectedDevices.Dec()

	}()
	logcl.Info("Client connected")

	// парсинг данных из протокола Arnavi
	concl := &ConClient{
		conn:          conn,
		authorization: false,
		log:           logcl,
		nc:            nc,
	}
	concl.process_connection()

}

type ConClient struct {
	conn          net.Conn
	authorization bool
	log           *logrus.Entry
	id            int // номер принятого пакета, если пакет отправлен то =0
	ImeiId        uint64
	nc            models.NatsConf
}

func (sc *ConClient) process_connection() {
	local_info := sc.conn.RemoteAddr()
	readBuf := make([]byte, 1024)
	localBuffer := new(bytes.Buffer)
	var (
		err error
		n   int
	)
	// sc.cursor = 0
	for {
		n, err = sc.conn.Read(readBuf)
		if err != nil {
			break
		}
		if n > 0 {
			sc.log.Debugf("dump:\n%s", hex.Dump(readBuf[:n]))
			localBuffer.Write(readBuf[:n])

			err = sc.processExistingData(localBuffer)

			if err != nil {
				sc.log.Infof("ошибка %v", err)
				break
			}
		}
	}
	sc.log.Infof("Disconnected from %s\n error: %v", local_info, err)

}
func (sc *ConClient) processExistingData(data *bytes.Buffer) error {
	var (
		err error
	)
	if !sc.authorization {
		return sc.Authorization(data)
	}

	for {
		buf := data.Bytes()
		nsize := data.Len()
		if nsize == 0 {
			return nil
		}
		if sc.id == 0 {
			if nsize < SizeScan {
				return nil
			}
			scan := ScanPaked{}
			if err = scan.Decode(buf); err != nil {
				return fmt.Errorf("не удалось просканировать scan %v", err)
			}
			sc.id = int(scan.Id)
			data.Next(SizeScan)
			continue
		}

		if buf[0] == SigPackEnd {
			if err = sc.finishpacked(); err != nil {
				return fmt.Errorf(" %v", err)
			}
			data.Next(1)
			sc.log.Debugf("отправлен  пакет подтверждения № %d", sc.id)
			sc.id = 0
			continue
		}
		if nsize < 3 {
			return err
		}
		scp := ScanPacket{}
		if err = scp.Decode(buf); err != nil {
			return fmt.Errorf("не удалось декодировать packet %v", err)
		}
		if nsize < (int(scp.LengthPacket) + 8) {
			return err
		}
		// запись пакета
		err = sc.savePacket(data)
		if err != nil {
			return err
		}
		data.Next(int(scp.LengthPacket) + 8)
	}
}
func (sc *ConClient) Authorization(data *bytes.Buffer) error {
	var (
		err error
	)
	nsize := data.Len()

	if nsize < SizeAuth {
		//ждем
		return err
	}
	headOne := HeadOne{}
	buf := data.Bytes()
	err = headOne.Decode(buf)
	if err != nil {
		return fmt.Errorf("ошибка разбора старт. пакета %v", err)
	}
	if headOne.Version == 0x24 {
		if nsize < (SizeAuth + 8) {
			return err
		}
		data.Next(SizeAuth + 8)
		sc.authorization = true
		sc.log.Debugf("Принят расширенный пакет авторизации Id|Imei: %d ", headOne.IdImei)

		return err
	}
	sc.ImeiId = headOne.IdImei
	sc.log = sc.log.WithField("imei_id", sc.ImeiId)
	sc.log.Debug("Принят пакет авторизации ")

	_, err = sc.conn.Write(AnswerHeader())
	if err != nil {
		sc.log.Errorf("ошибка отправки %v", err)
	}
	data.Next(SizeAuth)
	sc.authorization = true

	return err
}
func (sc *ConClient) finishpacked() error {
	var err error
	buf, err := AnswerPacked(sc.id)

	if err != nil {
		return fmt.Errorf("ошибка формирования подтверждения  %v", err)
	}
	if _, err := sc.conn.Write(buf); err != nil {
		return fmt.Errorf("ошибка формирования подтверждения  %v", err)
	}

	return err

}

func (sc *ConClient) savePacket(data *bytes.Buffer) error {
	var err error
	packets := PacketS{}
	buf := data.Bytes()
	if err = packets.Decode(buf); err != nil {
		return err
	}
	record := models.NavRecords{
		PacketID:   uint32(sc.id),
		PacketType: 2,
		RecNav:     make([]models.NavRecord, 1),
	}
	unixTime := time.Unix(int64(packets.TimePacket), 0)
	formattedTime := unixTime.Format("2006-01-02 15:04:05")
	switch pack := packets.Data.(type) {
	case *TagsData:
		record.RecNav[0].Imei = strconv.FormatUint(sc.ImeiId, 10)
		record.RecNav[0].NavigationTimestamp = packets.TimePacket
		record.RecNav[0].Latitude = uint32(pack.Latitude)
		record.RecNav[0].Longitude = uint32(pack.Longitude)
		// record.Speed = pack.Speed
		record.RecNav[0].Course = uint8(pack.Course)
		record.RecNav[0].ReceivedTimestamp = uint32(time.Now().Unix())
		record.RecNav[0].LiquidSensors.FlagLiqNum = 255
		record.RecNav[0].LiquidSensors.Value = pack.LL

		sc.log.Debugf(" %d получен пакет время: %s \n { time:%d, lat: %d, lon: %d }\n"+
			"course %d, speed %f, Satellites %X :, GPS %d, Glonass %d  LL: %v \n DATA: %v",
			sc.ImeiId,
			formattedTime, packets.TimePacket,
			pack.Latitude, pack.Longitude,
			pack.Course, pack.Speed, pack.Satellites, pack.Satellites&0xf, (pack.Satellites>>4)&0xf,
			pack.LL,
			pack.Data)

	default:
		sc.log.Infof("получен пакет неизвестный тип пакета № %X", packets.TypeContent)
		return fmt.Errorf("получен пакет неизвестный тип пакета № %X", packets.TypeContent)
	}
	js, _ := json.Marshal(record)

	_, err = sc.nc.Nc.Request(sc.nc.Topic, []byte(js), 2000*time.Millisecond)
	if err != nil {
		sc.log.Errorf("Nats error send: %v", err.Error())
		if sc.nc.Nc != nil {
			sc.nc.Nc.Close()
		}
		return err
	}

	return err
}
