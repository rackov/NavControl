package models

//go:generate easyjson -all -output_filename rest_api_json.go .
type ServiceModuleOld struct {
	IDSm        int    `json:"id_sm"`
	IPSm        string `json:"ip_sm"`
	Name        string `json:"name"`
	PortSm      int    `json:"port_sm"`
	TypeSm      string `json:"type_sm"`
	IsActive    bool   `json:"is_active"`
	Description string `json:"description"`
}

type ReceiverModuleOld struct {
	IDReceiver   int    `json:"id_receiver"`
	IDSm         int    `json:"id_sm"`
	IPReceiver   string `json:"ip_receiver"`
	IsActive     bool   `json:"is_active"`
	Name         string `json:"name"`
	KeyNats      string `json:"key_nats"`
	PortReceiver int    `json:"port_receiver"`
	Protocol     string `json:"protocol"`
	IPNats       string `json:"ip_nats"`
	PortNats     int    `json:"port_nats"`
	Status       string `json:"status"`
	Description  string `json:"description"`
}

type WriterModuleOld struct {
	IDWriter    int    `json:"id_writer"`
	IDSm        int    `json:"id_sm"`
	IPDB        string `json:"ip_db"`
	PortDB      int    `json:"port_db"`
	Name        string `json:"name"`
	IPNats      string `json:"ip_nats"`
	PortNats    int    `json:"port_nats"`
	KeyNats     string `json:"key_nats"`
	Status      string `json:"status"`
	IsActive    bool   `json:"is_active"`
	Description string `json:"description"`
	NameDB      string `json:"name_db"`
	TableDB     string `json:"table_db"`
	LoginDB     string `json:"login_db"`
	PasswDB     string `json:"passw_db"`
}

type RetranslatorModuleOld struct {
	IDRetranslator int      `json:"id_retranslator"`
	IDSm           int      `json:"id_sm"`
	IPClient       string   `json:"ip_client"`
	PortClient     int      `json:"port_client"`
	Protocol       string   `json:"protocol"`
	TimeStart      int      `json:"time_start"`
	Sensors        []string `json:"sensors"`
	DeviceList     []int    `json:"device_list"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	IsActive       bool     `json:"is_active"`
	Description    string   `json:"description"`
}

// новые структуры
type ServiceModule struct {
	IDSm        int    `json:"id_sm"`
	IPSm        string `json:"ip_sm"`
	PortSm      int    `json:"port_sm"`
	Name        string `json:"name"`
	TypeSm      string `json:"type_sm"`
	Active      bool   `json:"active"`
	Status      string `json:"status"`
	ErrorMsg    string `json:"error_msg"`
	Description string `json:"description"`
	LogLevel    string `json:"log_level"`
	IPBroker    string `json:"ip_broker"`
	PortBroker  int    `json:"port_broker"`
	TopicBroker string `json:"topic_broker"`
}
type TelematicProtocol struct {
	IdSm         int      `json:"id_sm"`
	ProtocolList []string `json:"protocol_list"`
}
type ReceiverModule struct {
	IDReceiver       int    `json:"id_receiver"`
	IDSm             int    `json:"id_sm"`
	Active           bool   `json:"active"`
	Name             string `json:"name"`
	PortReceiver     int    `json:"port_receiver"`
	Protocol         string `json:"protocol"`
	Status           string `json:"status"`
	ErrorMsg         string `json:"error_msg"`
	Description      string `json:"description"`
	ConnectionsCount int    `json:"connections_count"`
}
type IdInfo struct {
	Imei string `json:"imei"`
	Tid  int32  `json:"tid"`
}
type Clients struct {
	IDReceiver   int    `json:"id_receiver"`
	IDSm         int    `json:"id_sm"`
	ClientID     string `json:"client_id"`
	Address      string `json:"address"`
	ProtocolName string `json:"protocol_name"`
	PortReceiver int    `json:"port_receiver"`
	ConnectTime  int    `json:"connect_time"`
	LastTime     int    `json:"last_time"`
	CountPackets int    `json:"count_packets"`
	IdInfo       IdInfo `json:"number_device"`
	Multiple     bool   `json:"multiple"`
}

type WriterModule struct {
	IDWriter    int    `json:"id_writer"`
	IDSm        int    `json:"id_sm"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Active      bool   `json:"active"`
	Description string `json:"description"`
	NameDB      string `json:"name_db"`
	IPDB        string `json:"ip_db"`
	PortDB      int    `json:"port_db"`
	TableDB     string `json:"table_db"`
	LoginDB     string `json:"login_db"`
	PasswDB     string `json:"passw_db"`
}

type RetranslatorModule struct {
	IDRetranslator int      `json:"id_retranslator"`
	IDSm           int      `json:"id_sm"`
	IPClient       string   `json:"ip_client"`
	PortClient     int      `json:"port_client"`
	Protocol       string   `json:"protocol"`
	TimeStart      int      `json:"time_start"`
	Sensors        []string `json:"sensors"`
	DeviceList     []int    `json:"device_list"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	Active         bool     `json:"active"`
	Description    string   `json:"description"`
}

func NewToOldService(s *ServiceModule) *ServiceModuleOld {
	return &ServiceModuleOld{
		IDSm:        s.IDSm,
		IPSm:        s.IPSm,
		Name:        s.Name,
		PortSm:      s.PortSm,
		TypeSm:      s.TypeSm,
		IsActive:    s.Active,
		Description: s.Description,
	}
}

func NewToOldReceiver(r *ReceiverModule, s *ServiceModule) *ReceiverModuleOld {
	return &ReceiverModuleOld{
		IDReceiver:   r.IDReceiver,
		IDSm:         r.IDSm,
		IsActive:     r.Active,
		Name:         r.Name,
		KeyNats:      s.TopicBroker,
		PortReceiver: r.PortReceiver,
		Protocol:     r.Protocol,
		IPNats:       s.IPBroker,
		PortNats:     s.PortBroker,
		Status:       r.Status,
		Description:  r.Description,
	}
}
func NewToOldWrite(w *WriterModule, s *ServiceModule) *WriterModuleOld {
	return &WriterModuleOld{
		IDWriter:    w.IDWriter,
		IDSm:        w.IDSm,
		IPDB:        w.IPDB,
		PortDB:      w.PortDB,
		Name:        w.Name,
		IPNats:      s.IPBroker,
		PortNats:    s.PortBroker,
		KeyNats:     s.TopicBroker,
		Status:      w.Status,
		IsActive:    w.Active,
		Description: w.Description,
		NameDB:      w.NameDB,
		TableDB:     w.TableDB,
		LoginDB:     w.LoginDB,
		PasswDB:     w.PasswDB,
	}
}

func NewToOldRetranslator(r *RetranslatorModule) *RetranslatorModuleOld {
	return &RetranslatorModuleOld{
		IDRetranslator: r.IDRetranslator,
		IDSm:           r.IDSm,
		IPClient:       r.IPClient,
		PortClient:     r.PortClient,
		Protocol:       r.Protocol,
		TimeStart:      r.TimeStart,
		Sensors:        r.Sensors,
		DeviceList:     r.DeviceList,
		Name:           r.Name,
		Status:         r.Status,
		IsActive:       r.Active,
		Description:    r.Description,
	}
}

func OldToNewService(s *ServiceModuleOld, IpBroker string, PortBroker int, TopicBroker string) *ServiceModule {
	return &ServiceModule{
		IDSm:        s.IDSm,
		IPSm:        s.IPSm,
		PortSm:      s.PortSm,
		Name:        s.Name,
		TypeSm:      s.TypeSm,
		Active:      s.IsActive,
		Description: s.Description,
		IPBroker:    IpBroker,
		PortBroker:  PortBroker,
		TopicBroker: TopicBroker,
	}
}

type UsesMsgError struct {
	HttpCode   int    `json:"http_code"`
	ErrorTitle string `json:"error_title"`
	ErrorMsg   string `json:"error_msg"`
}
