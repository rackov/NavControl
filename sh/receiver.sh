#!/bin/bash

cd /home/vladimir/go/project/NavControl/services/receiver/cmd
go build -o /home/vladimir/go/project/NavControl/sh/bin/receiver *.go


# Параметры
USER="vladimir"           # Имя пользователя для SSH
REMOTE_HOST="192.168.30.165"   # IP-адрес или доменное имя удаленного компьютера
SERVICE_NAME="receiver" # Имя сервиса, который нужно остановить

# Подключение к удаленному серверу и остановка сервиса
ssh "$USER@$REMOTE_HOST" "sudo systemctl stop $SERVICE_NAME"

# Проверка успешности выполнения команды
if [ $? -eq 0 ]; then
    echo "Сервис $SERVICE_NAME успешно остановлен на $REMOTE_HOST."
else
    echo "Ошибка при остановке сервиса $SERVICE_NAME на $REMOTE_HOST."
fi

scp /home/vladimir/go/project/NavControl/sh/bin/receiver vladimir@$REMOTE_HOST:/home/vladimir/navcontrol/receiver
#scp /home/vladimir/go/project/NavControl/sh/bin/receiver.toml vladimir@192.168.194.242:/home/vladimir/navcontrol/receiver

ssh "$USER@$REMOTE_HOST" "sudo systemctl start $SERVICE_NAME"
if [ $? -eq 0 ]; then
    echo "Сервис $SERVICE_NAME успешно запущен на $REMOTE_HOST."
else
    echo "Ошибка при запуске сервиса $SERVICE_NAME на $REMOTE_HOST."
fi