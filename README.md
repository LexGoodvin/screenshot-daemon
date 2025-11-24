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


### Примеры использования:
### Несколько публичных сайтов:

ini
url = "https://google.com, https://github.com, https://stackoverflow.com"
time = "30m"
Микс публичных и защищенных:

ini
url = "https://example.com/public, https://example.com/private/dashboard"
login_url = "https://example.com/login"
login_username = "user"
login_password = "pass"
С разными протоколами:

ini
url = "https://site1.com, http://site2.com, https://site3.com/admin"
Особенности реализации:
Гибкий парсинг URL - поддерживает пробелы после запятых

Последовательная обработка - скриншоты делаются по очереди

Паузы между запросами - 2 секунды между скриншотами

Логирование прогресса - видно какой URL обрабатывается

Индивидуальная обработка ошибок - если один URL упал, остальные продолжают работать

Общая авторизация - логинимся один раз для всех URL в одной сессии