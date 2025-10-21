// pkg/arnavi/arnavi.go
package arnavi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
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

// id_imei
type IdInfo struct {
	Imei string `json:"imei"`
	Tid  int32  `json:"tid"`
}

// ClientInfo содержит информацию о подключенном клиенте
type ClientInfo struct {
	ID           string
	RemoteAddr   string
	ConnectTime  int32
	Protocol     string
	LastTime     int32
	Device       IdInfo
	CountPackets int64
	Multiple     bool
}
type ClientArnavi struct {
	conn          net.Conn
	authorization bool
	id            int // номер принятого пакета, если пакет отправлен то =0
	ImeiId        uint64
	clientID      string
	logger        *logrus.Entry
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

	logger *logrus.Entry
	nc     models.NatsConf

	// Добавим поля для JetStream
	js          jetstream.JetStream
	isJetStream bool
	//  таймер для проверки активности клиентов
	inactivityTimeout time.Duration
}

// NewArnaviProtocol создает новый экземпляр протокола Arnavi
func NewArnaviProtocol(log *logrus.Entry, isJetStream bool) protocol.NavigationProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	ep := &ArnaviProtocol{
		ctx:         ctx,
		cancel:      cancel,
		clients:     make(map[string]ClientInfo),
		connections: make(map[net.Conn]struct{}),
		logger:      log,
		isJetStream: isJetStream,
	}
	go ep.checkInactiveClients()
	return ep
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
	a.nc = nc

	// Инициализация JetStream если включено
	if a.isJetStream {
		js, err := jetstream.New(a.nc.Nc)
		if err != nil {
			a.logger.Errorf("Failed to initialize JetStream: %v", err)
			return err
		}
		a.js = js

		// Создание потока (stream) если он еще не существует
		_, err = js.CreateOrUpdateStream(a.ctx, jetstream.StreamConfig{
			Name:     "NAV_STREAM",
			Subjects: []string{a.nc.Topic},
			Storage:  jetstream.FileStorage,

			// MaxConsumers: 1,
		})

		if err != nil {
			a.logger.Errorf("Failed to create/update JetStream stream: %v", err)
			return err
		}

		a.logger.Info("JetStream initialized successfully")
	}
	a.logger.Infof("%s protocol started on port %d", a.GetName(), port)

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
				ConnectTime: int32(time.Now().Unix()),
				LastTime:    int32(time.Now().Unix()),
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

	concl := &ClientArnavi{
		conn:          conn,
		authorization: false,
		clientID:      clientID,
		id:            0,
		ImeiId:        0,
		logger:        logcl,
	}

	a.nc = nc
	a.process_connection(concl)

}

func (sc *ArnaviProtocol) process_connection(concl *ClientArnavi) {
	local_info := concl.conn.RemoteAddr()
	readBuf := make([]byte, 1024)
	localBuffer := new(bytes.Buffer)
	var (
		err error
		n   int
	)
	// sc.cursor = 0
	for {
		n, err = concl.conn.Read(readBuf)
		if err != nil {
			break
		}
		if n > 0 {
			concl.logger.Debugf("dump: %X", readBuf[:n])
			localBuffer.Write(readBuf[:n])

			err = sc.processExistingData(localBuffer, concl)

			if err != nil {
				concl.logger.Infof("ошибка %v", err)
				break
			}
		}
	}
	sc.logger.Infof("Disconnected from %s\n error: %v", local_info, err)

}
func (sc *ArnaviProtocol) processExistingData(data *bytes.Buffer, concl *ClientArnavi) error {
	var (
		err error
	)
	if !concl.authorization {
		return sc.Authorization(data, concl)
	}

	for {
		buf := data.Bytes()
		nsize := data.Len()
		if nsize == 0 {
			return nil
		}
		if concl.id == 0 {
			if nsize < SizeScan {
				return nil
			}
			scan := ScanPaked{}
			if err = scan.Decode(buf); err != nil {
				return fmt.Errorf("не удалось просканировать scan %v", err)
			}
			concl.id = int(scan.Id)
			concl.logger.Debug("Next scan")
			data.Next(SizeScan)
			continue
		}

		if buf[0] == SigPackEnd {
			if err = sc.finishpacked(concl); err != nil {
				return fmt.Errorf(" %v", err)
			}
			data.Next(1)
			concl.logger.Debugf("отправлен  пакет подтверждения № %d", concl.id)
			concl.id = 0
			continue
		}
		if nsize < 3 {
			return err
		}
		scp := ScanPacket{}
		if err = scp.Decode(buf); err != nil {
			return fmt.Errorf("не удалось декодировать packet %v", err)
		}
		concl.logger.Debugf("получен packet %d, динна пакета %d по протоколу %d, первый байт %X", scp.TypeContent, nsize, scp.LengthPacket, buf[0:1])
		if nsize < (int(scp.LengthPacket) + 8) {
			return err
		}
		// запись пакета
		err = sc.savePacket(data, concl)
		if err != nil {
			return err
		}
		data.Next(int(scp.LengthPacket) + 8)
	}
}
func (sc *ArnaviProtocol) Authorization(data *bytes.Buffer, concl *ClientArnavi) error {
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
		concl.authorization = true
		sc.logger.Debugf("Принят расширенный пакет авторизации Id|Imei: %d ", headOne.IdImei)

		return err
	}
	concl.ImeiId = headOne.IdImei
	concl.logger = concl.logger.WithField("imei_id", concl.ImeiId)
	concl.logger.Debug("Принят пакет авторизации ")

	_, err = concl.conn.Write(AnswerHeader())
	if err != nil {
		sc.logger.Errorf("ошибка отправки %v", err)
	}
	data.Next(SizeAuth)
	concl.authorization = true

	return err
}
func (sc *ArnaviProtocol) finishpacked(concl *ClientArnavi) error {
	var err error
	buf, err := AnswerPacked(concl.id)

	if err != nil {
		return fmt.Errorf("ошибка формирования подтверждения  %v", err)
	}
	if _, err := concl.conn.Write(buf); err != nil {
		return fmt.Errorf("ошибка формирования подтверждения  %v", err)
	}

	return err

}
func (eg *ArnaviProtocol) sendToNats(msg []byte) error {
	var err error

	eg.logger.Infof("sendToNats %s", msg)
	if eg.isJetStream {
		// Используем JetStream для публикации
		if eg.js == nil {
			err = fmt.Errorf("jetStream не инициализирован")
			return err
		}

		_, err = eg.js.Publish(eg.ctx, eg.nc.Topic, msg)
	} else {
		_, err = eg.nc.Nc.Request(eg.nc.Topic, msg, 2000*time.Millisecond)
	}
	return err
}

func (sc *ArnaviProtocol) savePacket(data *bytes.Buffer, concl *ClientArnavi) error {
	var err error
	packets := PacketS{}
	buf := data.Bytes()
	if err = packets.Decode(buf); err != nil {
		return err
	}
	record := models.NavRecords{
		PacketID:   uint32(concl.id),
		PacketType: 2,
		RecNav:     make([]models.NavRecord, 1),
	}

	switch pack := packets.Data.(type) {
	case *TagsData:
		record.RecNav[0].Imei = strconv.FormatUint(concl.ImeiId, 10)
		record.RecNav[0].NavigationTimestamp = packets.TimePacket
		record.RecNav[0].Latitude = uint32(pack.Latitude)
		record.RecNav[0].Longitude = uint32(pack.Longitude)
		record.RecNav[0].FlagPos = 129
		// record.Speed = pack.Speed
		record.RecNav[0].Course = uint8(pack.Course)
		record.RecNav[0].ReceivedTimestamp = uint32(time.Now().Unix())
		record.RecNav[0].LiquidSensors.FlagLiqNum = 255
		record.RecNav[0].LiquidSensors.Value = pack.LL

	default:
		concl.logger.Infof("получен пакет неизвестный тип пакета № %X", packets.TypeContent)
		return fmt.Errorf("получен пакет неизвестный тип пакета № %X", packets.TypeContent)
	}
	js, _ := json.Marshal(record)

	err = sc.sendToNats(js)
	if err != nil {
		sc.logger.Errorf("Nats error send: %v", err.Error())
		if sc.nc.Nc != nil {
			sc.nc.Nc.Close()
		}
		return err
	}
	concl.logger.Info("Данные: ", string(js))

	sc.clientsMu.Lock()
	if client, exists := sc.clients[concl.clientID]; exists {
		client.LastTime = int32(time.Now().Unix())
		client.Device = IdInfo{Tid: 0, Imei: record.RecNav[0].Imei}
		client.CountPackets++
		client.Multiple = false
		sc.clients[concl.clientID] = client
	}
	sc.clientsMu.Unlock()

	return err
}

// метод для проверки неактивных клиентов
func (a *ArnaviProtocol) checkInactiveClients() {
	ticker := time.NewTicker(1 * time.Minute) // Проверяем каждую минуту
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Unix()
			a.clientsMu.Lock()

			// Создаем список неактивных клиентов для отключения
			var inactiveClients []string
			for clientID, client := range a.clients {
				// Если прошло больше 3 минут с момента последней активности
				if (now - int64(client.LastTime)) > int64(a.inactivityTimeout.Seconds()) {
					inactiveClients = append(inactiveClients, clientID)
				}
			}

			a.clientsMu.Unlock()

			// Отключаем неактивных клиентов
			for _, clientID := range inactiveClients {
				a.DisconnectClient(clientID)
				a.logger.WithField("client", clientID).Info("Client disconnected due to inactivity")
			}
		}
	}
}
