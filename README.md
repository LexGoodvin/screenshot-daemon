# screenshot-daemon
make url screenshot by time and send in telegram ( linux daemon)
Install:
# Ubuntu/Debian
sudo apt update
sudo apt install chromium-browser

# CentOS/RHEL
sudo yum install chromium

Run as a systemd daemon
Create the file /etc/systemd/system/screenshot-daemon.service:

ini
[Unit]
Description=Screenshot Daemon
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/path/to/screenshot-daemon
ExecStart=/path/to/screenshot-daemon/screenshot-daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
Start the daemon:

bash
sudo systemctl daemon-reload
sudo systemctl enable screenshot-daemon
sudo systemctl start screenshot-daemon
Features
Uses Chrome Headless to create screenshots

Supports any web page

Automatically restarts on errors

Logs all actions

Easily configurable via INI file

Can be run as a system daemon.

The daemon will periodically take screenshots of the specified page and send them to the Telegram group according to the specified schedule.

Запуск как демона systemd
Создайте файл /etc/systemd/system/screenshot-daemon.service:

ini
[Unit]
Description=Screenshot Daemon
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/path/to/screenshot-daemon
ExecStart=/path/to/screenshot-daemon/screenshot-daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
Запустите демона:

bash
sudo systemctl daemon-reload
sudo systemctl enable screenshot-daemon
sudo systemctl start screenshot-daemon
Особенности
Использует Chrome Headless для создания скриншотов

Поддерживает любые веб-страницы

Автоматически перезапускается при ошибках

Логирует все действия

Легко настраивается через INI-файл

Можно запускать как системный демон

Демон будет периодически делать скриншоты указанной страницы и отправлять их в Telegram группу согласно заданному расписанию.
