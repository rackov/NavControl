# Сервис receiver
# Для проверки работы сервиса receiver необходимо запустить grpcurl
# Для отклика программы необходимо установить 
go get google.golang.org/grpc/reflection
# сделать изменения  в server.go
reflection.Register(s.grpcServer)

# Установите grpcurl, если еще не установлен: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 1. Добавляем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082,
  "protocol": "Arnavi",
  "active": false,
  "name": "Test Scenario Port"
}' localhost:50051 proto.ReceiverControl/AddPort

# 2. Открываем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/OpenPort

# 3. Проверяем количество подключений (должно быть 0)
grpcurl -plaintext -d '{
  "port_receiver": 8080
}' localhost:50051 proto.ReceiverControl/GetActiveConnectionsCount

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
  "client_id": "127.0.0.1:XXXXX-XXXXXXXXXX"
}' localhost:50051 proto.ReceiverControl/DisconnectClient

# 8. Закрываем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/ClosePort

# 9. Удаляем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/DeletePort
# 1. Добавляем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082,
  "protocol": "Arnavi",
  "active": false,
  "name": "Test Scenario Port"
}' localhost:50051 proto.ReceiverControl/AddPort

# 2. Открываем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/OpenPort

# 3. Проверяем количество подключений (должно быть 0)
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/GetActiveConnectionsCount

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
  "client_id": "127.0.0.1:XXXXX-XXXXXXXXXX"
}' localhost:50051 proto.ReceiverControl/DisconnectClient

# 8. Закрываем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/ClosePort

# 9. Удаляем порт
grpcurl -plaintext -d '{
  "port_receiver": 8082
}' localhost:50051 proto.ReceiverControl/DeletePort
