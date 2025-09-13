package sendegts

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/rackov/NavControl/services/receiver/handler/egts"

	log "github.com/sirupsen/logrus"
)

type SendPkg struct {
	Id_Gl    int
	Tid      int
	Imei     string
	Nav_Time int
	Lat      int
	Lng      int
	Speed    int
	Course   int
	Flag     int
	Din      int
	Asfe     int
	Ansi     [8]uint32
	Fl_Ln    int
	Val_L    [8]uint32
}

type SendParam struct {
	Ip       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type SendConnect struct {
	IdClient  int
	Active    bool
	Param     SendParam
	Field     []string
	netD      net.Conn
	isOpen    bool
	sigClient chan struct{}
	db        *sql.DB
	IdRec     int
	mutex     *sync.Mutex
}
type ArrConn []*SendConnect

func (sc *SendConnect) Run() {
	sc.isOpen = false
	sc.sigClient = make(chan struct{})
	log.Info("Запуск Id_client:", sc.IdClient)

	for {
		select {
		case <-sc.sigClient:
			log.Info("Остановка  Id_Client", sc.IdClient)
			if sc.netD != nil {
				sc.netD.Close()
			}
			sc.isOpen = false
			sc.sigClient = nil
			sc.Active = false
			return
		case <-time.After(200 * time.Millisecond):
			if err := sc.db.Ping(); err != nil {
				log.Info("ошибка подключения к базе:", sc.IdClient)
				sc.isOpen = false

				if sc.netD != nil {
					sc.netD.Close()
				}
				return
			}
			sc.sendRec()

		}
	}

}
func (sc *SendConnect) readPkg() ([]SendPkg, error) {
	var (
		err    error
		arrPkg []SendPkg
	)
	arrPkg = make([]SendPkg, 0)

	sql := fmt.Sprintf("from telematic.get_pkg_send(%d)", sc.IdClient)
	sql = "select id_gl,tid,imei,nav_time,lat,lng,speed,course," +
		"flag,din,asfe,ansi,fl_ln,val_l " + sql
	rows, err := sc.db.Query(sql)

	if err != nil {
		log.Info("Ошибка get_rows: ", err)
		return arrPkg, err
	}

	defer rows.Close()
	var ansi pq.Int64Array
	var val_l pq.Int64Array

	for rows.Next() {
		sendpkg := SendPkg{}
		err = rows.Scan(&sendpkg.Id_Gl, &sendpkg.Tid, &sendpkg.Imei,
			&sendpkg.Nav_Time, &sendpkg.Lat, &sendpkg.Lng, &sendpkg.Speed, &sendpkg.Course,
			&sendpkg.Flag,
			&sendpkg.Din, &sendpkg.Asfe, &ansi, &sendpkg.Fl_Ln, &val_l)
		if err != nil {
			log.Info("Ошибка считывания:", err)
			return arrPkg, err
		}
		for i, s := range ansi {
			sendpkg.Ansi[i] = uint32(s)
		}

		for i, s := range val_l {
			sendpkg.Val_L[i] = uint32(s)
		}

		arrPkg = append(arrPkg, sendpkg)
	}
	return arrPkg, err
}
func (sc *SendConnect) sendRec() {

	var send []int
	if !sc.isOpen {
		errd := sc.dial()
		if errd != nil {
			log.Info("error connect: ", sc.Param.Ip)
			return
		}
		sc.Active = true
	}

	send = make([]int, 0)
	arrPkg, err := sc.readPkg()

	for _, sendpkg := range arrPkg {

		ifequal := false
		err, ifequal = sc.received(sendpkg)
		if err == nil {
			if ifequal {
				send = append(send, sendpkg.Id_Gl)
			}
		} else {
			log.Info("Ошибка передачи ", err)
			sc.isOpen = false
			return
		}

		if len(send) == 0 {
			return
		} else {
			sc.save(send)
		}
	}
}
func (sc *SendConnect) received(sendpkg SendPkg) (err error, ifequal bool) {
	ifequal = false
	var RecordDataSet []egts.RecordData

	RecordDataSet = append(RecordDataSet,
		egts.RecordData{
			SubrecordType: egts.EGTS_SR_POS_DATA,
			SubrecordData: &egts.SrPosData{
				NavigationTime:      uint32(sendpkg.Nav_Time),
				Latitude:            uint32(sendpkg.Lat),
				Longitude:           uint32(sendpkg.Lng),
				FlagPos:             byte(sendpkg.Flag),
				DirectionHighestBit: 0,
				AltitudeSign:        0,
				Speed:               uint16(sendpkg.Speed),
				Direction:           byte(sendpkg.Course),
				Odometer:            0,
				DigitalInputs:       byte(sendpkg.Din),
				Source:              0,
				Altitude:            0,
			},
		})
	if sendpkg.Asfe > 0 {
		RecordDataSet = append(RecordDataSet,
			egts.RecordData{
				SubrecordType: egts.EGTS_SR_AD_SENSORS_DATA,
				SubrecordData: &egts.SrAdSensorsData{
					// DigitalInputsOctetExists:     0, //10011011
					// AdditionalDigitalInputsOctet: [8]byte{8, 7, 6, 5, 4, 3, 2, 1},
					// DigitalOutputs:               15,
					AnalogSensorFieldExists: byte(sendpkg.Asfe),
					AnalogSensors:           sendpkg.Ansi, //[8]uint32{8, 7, 6, 5, 4, 3, 2, 1},
				},
			},
		)
	}
	ServicesFrameData := egts.ServiceDataSet{}

	ServicesFrameData = append(ServicesFrameData,
		egts.ServiceDataRecord{
			RecordNumber:             uint16(sc.IdRec),
			SourceServiceOnDevice:    "0",
			RecipientServiceOnDevice: "0",
			Group:                    "0",
			RecordProcessingPriority: "10",
			TimeFieldExists:          "0",
			EventIDFieldExists:       "0",
			ObjectIDFieldExists:      "1",
			ObjectIdentifier:         uint32(sendpkg.Tid),
			SourceServiceType:        2,
			RecipientServiceType:     2,
			RecordDataSet:            RecordDataSet,
		},
	)

	egtsSendPacket := egts.Package{
		ProtocolVersion:   1,
		SecurityKeyID:     0,
		Prefix:            "00",
		Route:             "0",
		EncryptionAlg:     "00",
		Compression:       "0",
		Priority:          "10",
		HeaderLength:      11,
		HeaderEncoding:    0,
		PacketIdentifier:  uint16(sc.IdRec),
		PacketType:        1,
		ServicesFrameData: &ServicesFrameData,
	}

	sendBytes, err := egtsSendPacket.Encode()
	if err != nil {
		log.Info("Encode message failed: ", err)
		return err, ifequal
	}
	// fmt.Println("Send")
	err = sc.netD.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		log.Info("вышло время таймаута", err)
		return err, ifequal
	}
	_, err = sc.netD.Write(sendBytes)
	if err != nil {
		log.Info("Write to server failed:", err)
		return err, ifequal
	}
	readBuf := make([]byte, 1024)

	ackBuf := new(bytes.Buffer)

	for {
		n, err1 := sc.netD.Read(readBuf)
		if err1 != nil {
			log.Info("Disconnected ", err1)
			break
		}
		if n > 0 {
			term := false
			ackBuf.Write(readBuf[:n])
			term, ifequal = processExistingData(ackBuf, int(egtsSendPacket.PacketIdentifier))
			if term {
				break
			}

		} else {
			break
		}
	}
	if sc.IdRec <= 65535 {
		sc.IdRec = sc.IdRec + 1
	} else {
		sc.IdRec = 0
	}
	return err, ifequal
}
func processExistingData(data *bytes.Buffer, packId int) (term bool, ifequal bool) {
	ifequal = false
	for {
		if data.Len() < egts.HEADERLEN {
			return false, ifequal
		}
		if data.Bytes()[0] != 0x01 {
			// log.Info("Пакет не соответствует формату ЕГТС. Закрыто соединение")
			return true, ifequal
		}
		bodyLen := binary.LittleEndian.Uint16(data.Bytes()[5:7])
		pkgLen := uint16(data.Bytes()[3])
		if bodyLen > 0 {
			pkgLen += bodyLen + 2 //26
		}
		if data.Len() < int(pkgLen) {
			return false, ifequal
		}
		ackPacket := egts.Package{}

		_, err := ackPacket.Decode(data.Bytes()[:pkgLen])
		if err != nil {

			// log.Info("Parse ack packet failed:", err)
			data.Next(int(pkgLen))
			return false, ifequal
		}

		//	fmt.Println("type pkd ", ackPacket.PacketType)
		if egts.EGTS_PT_RESPONSE == ackPacket.PacketType {
			// fmt.Println("egts.EGTS_PT_RESPONSE")
			ack, ok := ackPacket.ServicesFrameData.(*egts.PtResponse)
			if !ok {
				// fmt.Println("Received packet is not egts ack")
				return false, ifequal
			}

			if ack.ResponsePacketID != uint16(packId) {
				// fmt.Printf(" Некоректный ответ ResponsePacketID: %d  != %d (id передано)",
				// 	ack.ResponsePacketID, packId)
				return false, ifequal
			} else {
				ifequal = true
				// fmt.Printf("Send ack.ResponsePacketID :%d\n", ack.ResponsePacketID)
				return true, ifequal
			}

		}
		data.Next(int(pkgLen))
		if data.Len() == 0 {
			return false, ifequal
		}
	}
}

func (sc *SendConnect) save(send_id []int) error {
	var err error
	mas, err := json.Marshal(&send_id)
	if err != nil {
		log.Info("Ошибка десериализации send ", err)
		return err
	}
	rows, err := sc.db.Query(`select * from telematic.set_pkg_send($1,$2)`, sc.IdClient, string(mas))
	if err != nil {
		log.Info("Ошибка записи подтверждения ", err)
		return err
	}
	rows.Close()
	return err
}

func (sc *SendConnect) Stop() {
	if sc.Active {
		// log.Info("signal to  client: ", sc.IdClient)
		close(sc.sigClient)

	}
	sc.Active = false
}

func (sc *SendConnect) dial() error {
	host := sc.Param.Ip
	port := sc.Param.Port
	sc.isOpen = false
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	sc.netD = conn
	sc.isOpen = true
	return err
}
