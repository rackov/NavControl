package mgrpc

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/writer/internal/rwnats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCServer struct {
	logger        *logger.Logger
	IpBroker      string
	PortBroker    int32
	TopicBroker   string
	SizeBuf       int
	IsIsJetStream bool
	wrNatsMap     map[int32]*rwnats.WrNatsServer
	Cfg           config.ControlWriter
	mu            sync.Mutex

	proto.UnimplementedServiceInfoServer
	proto.UnimplementedWriteControlServer
}

func NewGRPCServer(conf *config.ControlWriter, logger *logger.Logger) (*GRPCServer, error) {
	u, err := url.Parse(conf.NatsAddress)
	if err != nil {
		return nil, err
	}
	host := u.Host
	parts := strings.Split(host, ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	return &GRPCServer{
		IpBroker:      string(parts[0]),
		PortBroker:    int32(port),
		TopicBroker:   conf.NatsTopic,
		SizeBuf:       conf.SizeBuf,
		IsIsJetStream: conf.IsIsJetStream,
		logger:        logger,
		Cfg:           *conf,
		wrNatsMap:     make(map[int32]*rwnats.WrNatsServer),
	}, nil
}
func (s *GRPCServer) StartGRPCServer(port int) error {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		s.logger.Errorf("Failed to listen: %v", err)
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Регистрируем сервисы
	proto.RegisterServiceInfoServer(grpcServer, s)
	proto.RegisterWriteControlServer(grpcServer, s)

	// Включаем reflection API для отладки
	reflection.Register(grpcServer)

	for _, r := range s.Cfg.Writers {
		wrNats := &rwnats.WrNatsServer{
			IdWriter:    r.IdWriter,
			IdSm:        int32(s.Cfg.IdSm),
			IpDb:        r.DbIp,
			PortDb:      r.DbPort,
			DbName:      r.DbName,
			DbTable:     r.DbTable,
			Login:       r.DbUser,
			Passw:       r.DbPass,
			Name:        r.Name,
			Description: r.Description,
			IpBroker:    s.IpBroker,
			PortBroker:  s.PortBroker,
			TopicBroker: s.TopicBroker,
			Log:         s.logger,
			SizeBuf:     s.SizeBuf,
			IsJetStream: s.IsIsJetStream,
		}
		if err := wrNats.Start(); err != nil {
			s.logger.Errorf("Failed start Writer %d  err: %v", r.IdWriter, err)
		}
		s.logger.Debugf("Writer %v started", wrNats)
	}

	s.logger.Infof("gRPC server started on port %d", port)

	return grpcServer.Serve(lis)
}

// ListWrites returns a list of all write services
func (s *GRPCServer) ListWrites(ctx context.Context, _ *emptypb.Empty) (*proto.RetWrite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var writeServices []*proto.WriteService

	for _, wrNats := range s.wrNatsMap {
		writeService := &proto.WriteService{
			IdWriter:    wrNats.IdWriter,
			IdSm:        wrNats.IdSm,
			IpDb:        wrNats.IpDb,
			PortDb:      wrNats.PortDb,
			DbName:      wrNats.DbName,
			DbTable:     wrNats.DbTable,
			Login:       wrNats.Login,
			Passw:       wrNats.Passw,
			Status:      getStatusString(wrNats.DbConnected(), wrNats.NatsConnected()),
			Name:        wrNats.Name,
			IsActive:    wrNats.NatsConnected(),
			Description: wrNats.Description,
		}
		writeServices = append(writeServices, writeService)
	}

	return &proto.RetWrite{ListWrite: writeServices}, nil
}

// AddWrite adds a new write service
func (s *GRPCServer) AddWrite(ctx context.Context, writeService *proto.WriteService) (*proto.WriteService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if service already exists
	if _, exists := s.wrNatsMap[writeService.IdWriter]; exists {
		return nil, fmt.Errorf("write service already exists")
	}
	maxId := int32(0)
	for key, _ := range s.wrNatsMap {
		if key > maxId {
			maxId = key
		}
	}
	// Create new WrNatsServer instance
	wrNats := &rwnats.WrNatsServer{
		IdWriter:    writeService.IdWriter,
		IdSm:        writeService.IdSm,
		IpDb:        writeService.IpDb,
		PortDb:      writeService.PortDb,
		DbName:      writeService.DbName,
		DbTable:     writeService.DbTable,
		Login:       writeService.Login,
		Passw:       writeService.Passw,
		Name:        writeService.Name,
		Description: writeService.Description,
		IpBroker:    s.IpBroker,
		PortBroker:  s.PortBroker,
		TopicBroker: s.TopicBroker,
		Log:         s.logger,
		SizeBuf:     s.SizeBuf,
	}

	if err := wrNats.Start(); err != nil {
		return nil, err
	}

	// Add to map
	s.wrNatsMap[maxId+1] = wrNats

	s.Cfg.Writers = append(s.Cfg.Writers, config.Writer{
		IdWriter:    int32(maxId + 1),
		DbName:      writeService.DbName,
		DbTable:     writeService.DbTable,
		DbIp:        writeService.IpDb,
		DbPort:      int32(writeService.PortDb),
		DbUser:      writeService.Login,
		Description: writeService.Description,
		Name:        writeService.Name,
		DbPass:      writeService.Name,
	})

	s.Cfg.SaveConfig()

	return &proto.WriteService{
		IdWriter:    writeService.IdWriter,
		IdSm:        writeService.IdSm,
		IpDb:        writeService.IpDb,
		PortDb:      writeService.PortDb,
		Name:        writeService.Name,
		DbName:      writeService.DbName,
		DbTable:     writeService.DbTable,
		Login:       writeService.Login,
		Passw:       writeService.Passw,
		Status:      writeService.Status,
		IsActive:    writeService.IsActive,
		Description: writeService.Description,
	}, nil
}

// DeleteWrite removes an existing write service
func (s *GRPCServer) DeleteWrite(ctx context.Context, writeService *proto.WriteService) (*proto.WriteService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wrNats, exists := s.wrNatsMap[writeService.IdWriter]
	if !exists {
		return nil, fmt.Errorf("write service not found")
	}

	// Close the service
	wrNats.Close()

	// Remove from map
	delete(s.wrNatsMap, writeService.IdWriter)

	s.Cfg.DeleteWriter(writeService.IdWriter)

	return &proto.WriteService{
		IdWriter:    writeService.IdWriter,
		IdSm:        writeService.IdSm,
		IpDb:        writeService.IpDb,
		PortDb:      writeService.PortDb,
		Name:        writeService.Name,
		DbName:      writeService.DbName,
		DbTable:     writeService.DbTable,
		Login:       writeService.Login,
		Passw:       writeService.Passw,
		Status:      writeService.Status,
		IsActive:    writeService.IsActive,
		Description: writeService.Description,
	}, nil
}

// DownWrite disables a write service
func (s *GRPCServer) DownWrite(ctx context.Context, setWrite *proto.SetWrite) (*proto.WriteService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wrNats, exists := s.wrNatsMap[setWrite.IdWriter]
	if !exists {
		return nil, fmt.Errorf("write service not found")
	}

	// Close connections
	wrNats.Close()

	return &proto.WriteService{
		IdWriter:    wrNats.IdWriter,
		IdSm:        wrNats.IdSm,
		IpDb:        wrNats.IpDb,
		PortDb:      wrNats.PortDb,
		Name:        wrNats.Name,
		DbName:      wrNats.DbName,
		DbTable:     wrNats.DbTable,
		Login:       wrNats.Login,
		Passw:       wrNats.Passw,
		Status:      "closed",
		IsActive:    true,
		Description: wrNats.Description,
	}, nil
}

// UpWrite enables a write service
func (s *GRPCServer) UpWrite(ctx context.Context, setWrite *proto.SetWrite) (*proto.WriteService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wrNats, exists := s.wrNatsMap[setWrite.IdWriter]
	if !exists {
		return nil, fmt.Errorf("write service not found")
	}

	if err := wrNats.Start(); err != nil {
		return nil, err
	}

	return &proto.WriteService{
		IdWriter:    wrNats.IdWriter,
		IdSm:        wrNats.IdSm,
		IpDb:        wrNats.IpDb,
		PortDb:      wrNats.PortDb,
		Name:        wrNats.Name,
		DbName:      wrNats.DbName,
		DbTable:     wrNats.DbTable,
		Login:       wrNats.Login,
		Passw:       wrNats.Passw,
		Status:      "open",
		IsActive:    true,
		Description: wrNats.Description,
	}, nil
}

// Helper function to get status string based on connections
func getStatusString(dbConnected, natsConnected bool) string {
	if dbConnected && natsConnected {
		return "active"
	} else if natsConnected {
		return "nats_only"
	} else if dbConnected {
		return "db_only"
	}
	return "inactive"
}

// Logger and Service
// GetLogLevel возвращает текущий уровень логирования
func (s *GRPCServer) GetLogLevel(ctx context.Context, _ *emptypb.Empty) (*proto.LogLevelResponse, error) {
	level := s.logger.GetLevel()
	return &proto.LogLevelResponse{
		Level:   level,
		Success: true,
		Message: "Current log level: " + level,
	}, nil
}

// SetLogLevel устанавливает уровень логирования
func (s *GRPCServer) GetServiceManager(context.Context, *emptypb.Empty) (*proto.ServiceManager, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, err := url.Parse(s.Cfg.NatsAddress)
	if err != nil {
		return nil, err
	}
	host := u.Host
	parts := strings.Split(host, ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	return &proto.ServiceManager{
		Description: s.Cfg.Description,
		Name:        s.Cfg.Name,
		PortSm:      int32(s.Cfg.GrpcPort),
		TypeSm:      "RECEIVER",
		IpBroker:    string(parts[0]),
		PortBroker:  int32(port),
		TopicBroker: s.Cfg.NatsTopic,
		Active:      true,
		LogLevel:    s.Cfg.LogConfig.LogLevel,
	}, nil
}
func (s *GRPCServer) SetLogLevel(ctx context.Context, req *proto.SetLogLevelRequest) (*proto.SetLogLevelResponse, error) {
	err := s.logger.SetLevel(req.Level)
	if err != nil {
		s.logger.Errorf("Failed to set log level: %v", err)
		return &proto.SetLogLevelResponse{
			Success: false,
		}, nil
	}

	s.logger.Infof("Log level changed to %s", req.Level)
	return &proto.SetLogLevelResponse{
		Success: true,
	}, nil
}

// ReadLogs читает логи с применением фильтров
func (s *GRPCServer) ReadLogs(ctx context.Context, req *proto.ReadLogsRequest) (*proto.ReadLogsResponse, error) {
	// Получаем логи с фильтрами

	s.logger.Info("Reading logs")
	logReq := logger.Filter{
		Level:     req.Level,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		IdService: req.IdService,
		Protocol:  req.Protocol,
		Port:      req.Port,
		PosEnd:    req.PosEnd,
		Msg:       req.Msg,
	}
	logs, err := s.logger.ReadLogs(logReq)
	if err != nil {
		s.logger.Errorf("Failed to read logs: %v", err)
		return &proto.ReadLogsResponse{
			Success: false,
			Message: "Failed to read logs: " + err.Error(),
		}, nil
	}

	return &proto.ReadLogsResponse{
		LogLines: logs,
		Success:  true,
		Message:  "Logs retrieved successfully",
	}, nil
}
func (s *GRPCServer) GetInfo(ctx context.Context, req *emptypb.Empty) (*proto.ServiceInfoResponse, error) {
	return &proto.ServiceInfoResponse{
		Name:           "WRITER",
		Version:        "1.0.0",
		Build:          "1",
		BuildDate:      "2025-08-09",
		GitHash:        "https://github.com/rackov/NavControl.git",
		GitBranch:      "master",
		GitState:       "clean",
		GitSummary:     "clean",
		GitDescription: "clean",
		GitUrl:         "https://github.com/rackov/NavControl.git",
		GitUser:        "rackov",
		GitEmail:       "rackov@gmail.com",
		GitRemote:      "https://github.com/rackov/NavControl.git",
	}, nil
}
