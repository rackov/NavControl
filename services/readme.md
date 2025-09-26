# Сборка образа для receiver сервиса
docker build -t receiver-service -f services/receiver/Dockerfile .

# Сборка образа для rest-api сервиса
docker build -t rest-api-service -f services/rest-api/Dockerfile .

# Экспорт receiver сервиса в tar файл
docker save receiver-service | gzip > receiver-service.tar.gz

# Экспорт rest-api сервиса в tar файл
docker save rest-api-service | gzip > rest-api-service.tar.gz


# Импорт receiver сервиса
docker load -i receiver-service.tar.gz

# Импорт rest-api сервиса
docker load -i rest-api-service.tar.gz

# Запуск receiver сервиса
docker run -d -p 8080:8080 -p 9090:9090 --name receiver \
  -v /home/vladimir/go/project/NavControl/cfg:/app/NavControl/cfg \
  -v /home/vladimir/go/project/NavControl/log:/log \
  receiver-service

# Запуск rest-api сервиса
docker run -p 8082:8082 -p 9990:9990  --name rest-api \
  -v /home/vladimir/go/project/NavControl/cfg:/app/NavControl/cfg \
  -v /home/vladimir/go/project/NavControl/logs:/logs \
  rest-api-service