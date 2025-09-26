#                              rest-api.service
# Создайте файл службы:
sudo nano /etc/systemd/system/rest-api.service
# Добавьте в него следующее содержание:
rest-api.service
# Перезагрузите systemd и запустите службу:
sudo systemctl daemon-reload
sudo systemctl start rest-api
sudo systemctl enable rest-api
# Проверьте статус службы
sudo systemctl status rest-api
# Настройте брандмауэр (опционально)
sudo ufw allow 80/tcp   # Пример: разрешить HTTP
sudo ufw allow 443/tcp  # Пример: разрешить HTTPS


#                              receiver.service
# Создайте файл службы:
sudo nano /etc/systemd/system/receiver.service
# Добавьте в него следующее содержание:
receiver.service
# Перезагрузите systemd и запустите службу:
sudo systemctl daemon-reload
sudo systemctl start receiver
sudo systemctl enable receiver
# Проверьте статус службы
sudo systemctl status receiver

#                         retranslator.service
# Создайте файл службы:
sudo nano /etc/systemd/system/retranslator.service
# Добавьте в него следующее содержание:
retranslator.service
# Перезагрузите systemd и запустите службу:
sudo systemctl daemon-reload
sudo systemctl start retranslator
sudo systemctl enable retranslator
# Проверьте статус службы
sudo systemctl status retranslator

#                          writer.service
# Создайте файл службы:
sudo nano /etc/systemd/system/writer.service
# Добавьте в него следующее содержание:
writer.service
# Перезагрузите systemd и запустите службу:
sudo systemctl daemon-reload
sudo systemctl start writer
sudo systemctl enable writer
# Проверьте статус службы
sudo systemctl status writer

#                         webcurl
# Создайте файл службы:
sudo nano /etc/systemd/system/webcurl.service
# Добавьте в него следующее содержание:
webcurl.service
# Перезагрузите systemd и запустите службу:
sudo systemctl daemon-reload
sudo systemctl start webcurl
sudo systemctl enable webcurl
# Проверьте статус службы
sudo systemctl status webcurl