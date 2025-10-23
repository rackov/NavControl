# Сервис receiver
# Для проверки работы сервиса receiver необходимо запустить grpcurl
# Для отклика программы необходимо установить 
go get google.golang.org/grpc/reflection
# сделать изменения  в server.go
reflection.Register(s.grpcServer)

# Установите grpcurl, если еще не установлен: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 0.1 Сведения о сервисе
grpcurl -plaintext localhost:50004 proto.ServiceInfo/GetInfo

# 0.2  Сведения о сервис-менеджере
grpcurl -plaintext localhost:50004 proto.ServiceInfo/GetServiceManager

# 1
grpcurl -plaintext localhost:50004 proto.WriteControl/ListWrites

# 1. Добавляем порт
grpcurl -plaintext -d '{
  "port_receiver": 8085,
  "protocol": "Arnavi",
  "active": true,
  "name": "Test Scenario Port",
  "description": "This is a test scenario port"
}' localhost:50051 proto.ReceiverControl/AddPort

# 2. Открываем порт
grpcurl -plaintext -d '{
  "port_receiver": 8982
}' localhost:50000 proto.ReceiverControl/OpenPort

# 3. Проверяем количество подключений (должно быть 0)
grpcurl -plaintext -d '{
  "port_receiver": 8982
}' localhost:50000 proto.ReceiverControl/GetActiveConnectionsCount

# 4. Подключаемся к порту 8082 с помощью telnet в другом терминале:
# telnet localhost 8082

# 5. Снова проверяем количество подключений (должно быть 1)
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/GetActiveConnectionsCount

# 6. Получаем список клиентов
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/GetConnectedClients

# 7. Отключаем клиента (используем ID из предыдущего запроса)
grpcurl -plaintext -d '{
  "client_id": "127.0.0.1:45062-1756998166"
}' localhost:50051 proto.ReceiverControl/DisconnectClient

# 8. Закрываем порт
grpcurl -plaintext -d '{
  "port_receiver": 8982
}' localhost:50051 proto.ReceiverControl/ClosePort

# 9. Удаляем порт
grpcurl -plaintext -d '{
  "port_receiver": 8983
}' localhost:50000 proto.ReceiverControl/DeletePort

# 10  Список портов
grpcurl -plaintext localhost:50051 proto.ReceiverControl/ListPorts

# 11. Статус порта
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/GetPortStatus

grpcurl -plaintext -d '{
    "limit": 100
}' localhost:50051 proto.LoggingControl/ReadLogs

grpcurl -plaintext -d '{
  "level": "info",
  "start_date": '$(date -d '1 hour ago' +%s)',
  "end_date": '$(date +%s)',
  "limit": 100
}' localhost:50051 proto.LoggingControl/ReadLogs

grpcurl -plaintext -d '{
  "level": "",
  "start_date": 0,
  "end_date": 0,
  "limit": 5
}' localhost:50000 proto.ServiceInfo/ReadLogs

grpcurl -plaintext -d '{
  "level": "INFO",
  "limit": 100
}' localhost:50051 proto.LoggingControl/ReadLogs

grpcurl -plaintext -d '{
    "level": "DEBUG"
}' localhost:50051 proto.LoggingControl/SetLogLevel


# 1. Добавляем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082,
  "protocol": "EGTS",
  "is_active": true,
  "name": "Test Scenario Port"
}' localhost:50051 proto.ReceiverControl/AddPort

