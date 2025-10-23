#!/bin/bash

cd /home/vladimir/go/project/NavControl/services/writer/cmd
go build -o /home/vladimir/go/project/NavControl/sh/bin/writer *.go


# Параметры
USER="vladimir"           # Имя пользователя для SSH
REMOTE_HOST="192.168.194.242"   # IP-адрес или доменное имя удаленного компьютера
SERVICE_NAME="writer" # Имя сервиса, который нужно остановить

# Подключение к удаленному серверу и остановка сервиса
ssh "$USER@$REMOTE_HOST" "sudo systemctl stop $SERVICE_NAME"

# Проверка успешности выполнения команды
if [ $? -eq 0 ]; then
    echo "Сервис $SERVICE_NAME успешно остановлен на $REMOTE_HOST."
else
    echo "Ошибка при остановке сервиса $SERVICE_NAME на $REMOTE_HOST."
fi

scp /home/vladimir/go/project/NavControl/sh/bin/writer vladimir@$REMOTE_HOST:/home/vladimir/navcontrol/writer
#scp /home/vladimir/go/project/NavControl/sh/bin/writer.toml vladimir@192.168.194.242:/home/vladimir/navcontrol/writer

ssh "$USER@$REMOTE_HOST" "sudo systemctl start $SERVICE_NAME"
if [ $? -eq 0 ]; then
    echo "Сервис $SERVICE_NAME успешно запущен на $REMOTE_HOST."
else
    echo "Ошибка при запуске сервиса $SERVICE_NAME на $REMOTE_HOST."
fi