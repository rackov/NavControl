package egts

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/services/receiver/internal/protocol"
	"github.com/sirupsen/logrus"
)

const (
	egtsPcOk  = 0
	headerLen = 10
)

var (
	connectedDevices1 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "navcontrol_connected_egts_total",
			Help: "Total number of currently connected devices",
		},
	)
)

func init() {
	// Регистрация метрики в Prometheus
	prometheus.MustRegister(connectedDevices1)
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

// EgtsProtocol реализует интерфейс NavigationProtocol для протокола Arnavi
type EgtsProtocol struct {
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
	clients     map[string]ClientInfo
	clientsMu   sync.Mutex
	connections map[net.Conn]struct{}
	connMu      sync.Mutex
	logger      *logrus.Entry
	// Egts клиент
	conn      net.Conn
	isPkgSave bool
	nc        models.NatsConf
}

// NewEgtsProtocol создает новый экземпляр протокола Arnavi
func NewEgtsProtocol(log *logrus.Entry) protocol.NavigationProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	return &EgtsProtocol{
		ctx:         ctx,
		cancel:      cancel,
		clients:     make(map[string]ClientInfo),
		connections: make(map[net.Conn]struct{}),
		logger:      log,
	}
}

// GetName возвращает имя протокола
func (a *EgtsProtocol) GetName() string {
	return "EGTS"
}

// Start запускает TCP-сервер для приема данных
func (a *EgtsProtocol) Start(port int, nc models.NatsConf) error {
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
func (a *EgtsProtocol) handleConnections(nc models.NatsConf) {
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
			connectedDevices1.Inc()
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
func (a *EgtsProtocol) Stop() error {
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
func (a *EgtsProtocol) GetClients() []ClientInfo {
	a.clientsMu.Lock()
	defer a.clientsMu.Unlock()

	clients := make([]ClientInfo, 0, len(a.clients))
	for _, client := range a.clients {
		clients = append(clients, client)
	}

	return clients
}

// DisconnectClient отключает клиента по ID
func (a *EgtsProtocol) DisconnectClient(clientID string) error {
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
func (a *EgtsProtocol) handleConnection(conn net.Conn, clientID string, nc models.NatsConf) {
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

		connectedDevices1.Dec()

	}()
	logcl.Info("Client connected")

	// парсинг данных из протокола Arnavi

	a.conn = conn
	a.isPkgSave = true
	a.logger = logcl
	a.nc = nc

	a.process_connection(clientID)

}

// type ConClient struct {
// 	conn      net.Conn
// 	isPkgSave bool
// 	nc        models.NatsConf
// 	log       *logrus.Entry
// }

func (eg *EgtsProtocol) process_connection(clientID string) {

	var (
		srResultCodePkg   []byte
		serviceType       uint8
		srResponsesRecord RecordDataSet
		recvPacket        []byte
		client            uint32
	)
	local := eg.conn
	sttl := 1 * time.Second
	connTimer := time.NewTimer(sttl)

	for {
	Received:
		serviceType = 0
		srResponsesRecord = nil
		srResultCodePkg = nil
		recvPacket = nil
		connTimer.Reset(sttl)

		// считываем заголовок пакета
		headerBuf := make([]byte, headerLen)

		_, err := io.ReadFull(local, headerBuf)

		switch err {
		case nil:
			connTimer.Reset(sttl)

			// если пакет не егтс формата закрываем соединение
			if headerBuf[0] != 0x01 {
				eg.logger.Warn("Пакет не соответствует формату ЕГТС. Закрыто соединение")
				return
			}

			// вычисляем длину пакета, равную длине заголовка (HL) + длина тела (FDL) + CRC пакета 2 байта если есть FDL из приказа минтранса №285
			bodyLen := binary.LittleEndian.Uint16(headerBuf[5:7])
			pkgLen := uint16(headerBuf[3])
			if bodyLen > 0 {
				pkgLen += bodyLen + 2
			}
			// получаем концовку ЕГТС пакета
			buf := make([]byte, pkgLen-headerLen)
			_, err := io.ReadFull(local, buf)

			if err != nil {
				eg.logger.WithField("err", err).Error("Ошибка при получении тела пакета")
				return
			}

			// формируем полный пакет
			recvPacket = append(headerBuf, buf...)
		case io.EOF:
			<-connTimer.C
			eg.logger.Info("Соединение закрыто ")
			return
		default:
			//	local.Close()
			// log.WithField("err", err).Error("Ошибка при получении")
			return
		}

		eg.logger.WithField("packet", fmt.Sprintf("%x", recvPacket)).Debug("Принят пакет")
		pkg := Package{}
		receivedTimestamp := time.Now().UTC().Unix()
		resultCode, err := pkg.Decode(recvPacket)
		if err != nil {
			eg.logger.WithField("err", err).Error("Ошибка расшифровки пакета")

			resp, err := createPtResponse(pkg.PacketIdentifier, resultCode, serviceType, nil)
			if err != nil {
				eg.logger.WithField("err", err).Error("Ошибка сборки ответа EGTS_PT_RESPONSE с ошибкой")
				goto Received
			}
			_, _ = local.Write(resp)

			goto Received
		}
		switch pkg.PacketType {
		case EGTS_PT_APPDATA:
			eg.logger.Debug("Тип пакета EGTS_PT_APPDATA")

			for _, rec := range *pkg.ServicesFrameData.(*ServiceDataSet) {
				// создаем переменную выгрузки

				packetIDBytes := make([]byte, 4)

				srResponsesRecord = append(srResponsesRecord, RecordData{
					SubrecordType:   EGTS_SR_RECORD_RESPONSE,
					SubrecordLength: 3,
					SubrecordData: &SrResponse{
						ConfirmedRecordNumber: rec.RecordNumber,
						RecordStatus:          egtsPcOk,
					},
				})

				exportPackets := models.NavRecords{
					PacketType: 1,
					PacketID:   uint32(rec.RecordNumber),
				}

				serviceType = rec.SourceServiceType

				// если в секции с данными есть oid то обновляем его
				if rec.ObjectIDFieldExists == "1" {
					client = rec.ObjectIdentifier
				}
				i := 0
				Imei := ""
				Imsi := ""
				for _, subRec := range rec.RecordDataSet {
					switch subRecData := subRec.SubrecordData.(type) {
					case *SrTermIdentity:
						eg.logger.Debug("Разбор подзаписи EGTS_SR_TERM_IDENTITY")

						// на случай если секция с данными не содержит oid
						client = subRecData.TerminalIdentifier

						Imei = subRecData.IMEI
						Imsi = subRecData.IMSI
						if srResultCodePkg, err = createSrResultCode(pkg.PacketIdentifier, egtsPcOk); err != nil {
							eg.logger.Errorf("Ошибка сборки EGTS_SR_RESULT_CODE: %v", err)
						}
					case *SrAuthInfo:
						eg.logger.Debug("Разбор подзаписи EGTS_SR_AUTH_INFO")
						if srResultCodePkg, err = createSrResultCode(pkg.PacketIdentifier, egtsPcOk); err != nil {
							eg.logger.Errorf("Ошибка сборки EGTS_SR_RESULT_CODE: %v", err)
						}
					case *SrResponse:
						eg.logger.Debugf("Разбор подзаписи EGTS_SR_RESPONSE")
						goto Received
					case *SrPosData:
						eg.logger.Debugf("Разбор подзаписи EGTS_SR_POS_DATA")

						exportPackets.RecNav = append(exportPackets.RecNav, models.NavRecord{})
						exportPackets.RecNav[i].Imei = Imei
						exportPackets.RecNav[i].Imsi = Imsi
						exportPackets.RecNav[i].PacketID = uint32(i + 1)
						exportPackets.RecNav[i].Client = client
						exportPackets.RecNav[i].NavigationTimestamp = subRecData.NavigationTime //.Unix()
						exportPackets.RecNav[i].ReceivedTimestamp = uint32(receivedTimestamp)
						exportPackets.RecNav[i].Latitude = subRecData.Latitude
						exportPackets.RecNav[i].Longitude = subRecData.Longitude
						exportPackets.RecNav[i].Speed = subRecData.Speed
						exportPackets.RecNav[i].Course = subRecData.Direction
						exportPackets.RecNav[i].FlagPos = subRecData.FlagPos
						exportPackets.RecNav[i].DigInput = subRecData.DigitalInputs
						exportPackets.RecNav[i].Odometer = subRecData.Odometer

						i = i + 1 // обязательный пакет
					case *SrExtPosData:
						k := 0
						if i > 0 {
							k = i - 1
						} else if i == 0 {
							exportPackets.RecNav = append(exportPackets.RecNav, models.NavRecord{})
						}
						eg.logger.Debug("Разбор подзаписи EGTS_SR_EXT_POS_DATA")
						exportPackets.RecNav[k].Nsat = subRecData.Satellites
						exportPackets.RecNav[k].Pdop = subRecData.PositionDilutionOfPrecision
						exportPackets.RecNav[k].Hdop = subRecData.HorizontalDilutionOfPrecision
						exportPackets.RecNav[k].Vdop = subRecData.VerticalDilutionOfPrecision
						exportPackets.RecNav[k].Ns = subRecData.NavigationSystem

					case *SrAdSensorsData: //EGTS_SR_AD_SENSORS_DATA
						k := 0
						if i > 0 {
							k = i - 1
						} else if i == 0 {
							exportPackets.RecNav = append(exportPackets.RecNav, models.NavRecord{})
						}
						eg.logger.Debug("Разбор подзаписи EGTS_SR_AD_SENSORS_DATA")
						exportPackets.RecNav[k].DigSenOuts = append(exportPackets.RecNav[k].DigSenOuts, int(subRecData.DigitalOutputs))
						exportPackets.RecNav[k].DigSenonrs = append(exportPackets.RecNav[k].DigSenonrs, models.DopDigIn{Dioe: subRecData.DigitalInputsOctetExists, Adio: subRecData.AdditionalDigitalInputsOctet})

						exportPackets.RecNav[k].AnSensors = append(exportPackets.RecNav[k].AnSensors, models.DopAnIn{Asfe: subRecData.AnalogSensorFieldExists, Ansi: subRecData.AnalogSensors})

					case *SrAbsAnSensData:
						k := 0
						if i > 0 {
							k = i - 1
						} else if i == 0 {
							exportPackets.RecNav = append(exportPackets.RecNav, models.NavRecord{})
						}
						eg.logger.Debug("Разбор подзаписи EGTS_SR_ABS_AN_SENS_DATA")
						exportPackets.RecNav[k].AnSenAbs = append(exportPackets.RecNav[k].AnSenAbs, models.Sensor{SensorNumber: subRecData.SensorNumber, Value: subRecData.Value})
					case *SrAbsDigSensData:
						k := 0
						if i > 0 {
							k = i - 1
						} else if i == 0 {
							exportPackets.RecNav = append(exportPackets.RecNav, models.NavRecord{})
						}
						eg.logger.Debug("Разбор подзаписи EGTS_SR_ABS_DIG_SENS_DATA")
						exportPackets.RecNav[k].DigSenAbs = append(exportPackets.RecNav[k].DigSenAbs, models.DiSensor{StateNumber: subRecData.StateNumber, Number: subRecData.Number})

					case *SrAbsCntrData:
						k := 0
						if i > 0 {
							k = i - 1
						} else if i == 0 {
							exportPackets.RecNav = append(exportPackets.RecNav, models.NavRecord{})
						}
						eg.logger.Debug("Разбор подзаписи EGTS_SR_ABS_CNTR_DATA")

						switch subRecData.CounterNumber {
						case 110:
							// Три младших байта номера передаваемой записи (идет вместе с каждой POS_DATA).
							binary.BigEndian.PutUint32(packetIDBytes, subRecData.CounterValue)
							exportPackets.RecNav[k].PacketID = subRecData.CounterValue
						case 111:
							// один старший байт номера передаваемой записи (идет вместе с каждой POS_DATA).
							tmpBuf := make([]byte, 4)
							binary.BigEndian.PutUint32(tmpBuf, subRecData.CounterValue)

							if len(packetIDBytes) == 4 {
								packetIDBytes[3] = tmpBuf[3]
							} else {
								packetIDBytes = tmpBuf
							}

							exportPackets.RecNav[k].PacketID = binary.LittleEndian.Uint32(packetIDBytes)
						}
					case *SrLiquidLevelSensor:
						k := 0
						if i > 0 {
							k = i - 1
						} else if i == 0 {
							exportPackets.RecNav = append(exportPackets.RecNav, models.NavRecord{})
						}
						eg.logger.Debug("Разбор подзаписи EGTS_SR_LIQUID_LEVEL_SENSOR")
						ind := (subRecData.FlagLiq & 7)
						offset := 1 << ind
						exportPackets.RecNav[k].LiquidSensors.FlagLiqNum = exportPackets.RecNav[k].LiquidSensors.FlagLiqNum | uint8(offset)
						exportPackets.RecNav[k].LiquidSensors.Value[ind] = subRecData.LiquidLevelSensorData
						eg.logger.Debug("Разбор подзаписи EGTS_SR_LIQUID_LEVEL_SENSOR N", ind, " TID: ", client, "Lev", subRecData.LiquidLevelSensorData)
					}
				}

				if eg.isPkgSave {
					js, _ := json.Marshal(exportPackets)

					_, err := eg.nc.Nc.Request(eg.nc.Topic, []byte(js), 2000*time.Millisecond)
					if err != nil {
						eg.logger.Errorf("Nats error send:  %s", err.Error())
						if eg.nc.Nc != nil {
							eg.nc.Nc.Close()
						}
						return
					}
					eg.logger.Info("Данные: ", string(js))
					l := len(exportPackets.RecNav)

					eg.clientsMu.Lock()
					if client, exists := eg.clients[clientID]; exists {
						client.LastTime = int32(time.Now().Unix())
						client.Multiple = false
						if l == 1 {
							client.Device = IdInfo{Tid: int32(exportPackets.RecNav[0].Client), Imei: exportPackets.RecNav[0].Imei}
							client.Multiple = true
						}
						client.CountPackets++

						eg.clients[clientID] = client
					}
					eg.clientsMu.Unlock()
				}
			}

			resp, err := createPtResponse(pkg.PacketIdentifier, resultCode, serviceType, srResponsesRecord)
			if err != nil {
				eg.logger.WithField("err", err).Error("Ошибка сборки ответа")
				goto Received
			}
			_, _ = local.Write(resp)

			eg.logger.WithField("packet", resp).Debug("Отправлен пакет EGTS_PT_RESPONSE")

			if len(srResultCodePkg) > 0 {
				_, _ = local.Write(srResultCodePkg)
				eg.logger.WithField("packet", resp).Debug("Отправлен пакет EGTS_SR_RESULT_CODE")
			}
		case EGTS_PT_RESPONSE:
			eg.logger.Debug("Тип пакета EGTS_PT_RESPONSE")
		}
	}

}

func createPtResponse(pid uint16, resultCode, serviceType uint8, srResponses RecordDataSet) ([]byte, error) {
	respSection := PtResponse{
		ResponsePacketID: pid,
		ProcessingResult: resultCode,
	}

	if srResponses != nil {
		respSection.SDR = &ServiceDataSet{
			ServiceDataRecord{
				RecordLength:             srResponses.Length(),
				RecordNumber:             1,
				SourceServiceOnDevice:    "0",
				RecipientServiceOnDevice: "0",
				Group:                    "1",
				RecordProcessingPriority: "00",
				TimeFieldExists:          "0",
				EventIDFieldExists:       "0",
				ObjectIDFieldExists:      "0",
				SourceServiceType:        serviceType,
				RecipientServiceType:     serviceType,
				RecordDataSet:            srResponses,
			},
		}
	}

	respPkg := Package{
		ProtocolVersion:   1,
		SecurityKeyID:     0,
		Prefix:            "00",
		Route:             "0",
		EncryptionAlg:     "00",
		Compression:       "0",
		Priority:          "00",
		HeaderLength:      11,
		HeaderEncoding:    0,
		FrameDataLength:   respSection.Length(),
		PacketIdentifier:  pid + 1,
		PacketType:        EGTS_PT_RESPONSE,
		ServicesFrameData: &respSection,
	}

	return respPkg.Encode()
}

func createSrResultCode(pid uint16, resultCode uint8) ([]byte, error) {
	rds := RecordDataSet{
		RecordData{
			SubrecordType:   EGTS_SR_RESULT_CODE,
			SubrecordLength: uint16(1),
			SubrecordData: &SrResultCode{
				ResultCode: resultCode,
			},
		},
	}

	sfd := ServiceDataSet{
		ServiceDataRecord{
			RecordLength:             rds.Length(),
			RecordNumber:             1,
			SourceServiceOnDevice:    "0",
			RecipientServiceOnDevice: "0",
			Group:                    "1",
			RecordProcessingPriority: "00",
			TimeFieldExists:          "0",
			EventIDFieldExists:       "0",
			ObjectIDFieldExists:      "0",
			SourceServiceType:        SERVICE_AUTH,
			RecipientServiceType:     SERVICE_AUTH,
			RecordDataSet:            rds,
		},
	}

	respPkg := Package{
		ProtocolVersion:   1,
		SecurityKeyID:     0,
		Prefix:            "00",
		Route:             "0",
		EncryptionAlg:     "00",
		Compression:       "0",
		Priority:          "00",
		HeaderLength:      11,
		HeaderEncoding:    0,
		FrameDataLength:   sfd.Length(),
		PacketIdentifier:  pid + 1,
		PacketType:        EGTS_PT_APPDATA, // EGTS_PT_RESPONSE,
		ServicesFrameData: &sfd,
	}

	return respPkg.Encode()
}
