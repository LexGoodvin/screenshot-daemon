# Screenshot Daemon / Демон для скриншотов

![Systemd](https://img.shields.io/badge/systemd-daemon-blue)
![Telegram](https://img.shields.io/badge/telegram-bot-blue)
![Chromium](https://img.shields.io/badge/chromium-headless-green)

A Linux daemon that takes screenshots of web pages at specified intervals and sends them to Telegram.
Linux демон, который делает скриншоты веб-страниц через заданные интервалы и отправляет их в Telegram.

## Features / Особенности
- ✅ Uses Chrome Headless for screenshots / Использует Chrome Headless для создания скриншотов
- ✅ Supports any web page / Поддерживает любые веб-страницы
- ✅ Automatically restarts on errors / Автоматически перезапускается при ошибках
- ✅ Logs all actions / Логирует все действия
- ✅ Easily configurable via INI file / Легко настраивается через INI-файл
- ✅ Can be run as a system daemon / Можно запускать как системный демон

## Installation / Установка

### Install Chromium / Установите Chromium

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install chromium-browser
