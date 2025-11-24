package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/go-ini/ini"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ScreenConfig struct {
	Width  int
	Height int
}

type LoginConfig struct {
	URL              string
	Username         string
	Password         string
	UsernameSelector string
	PasswordSelector string
	SubmitSelector   string
	WaitAfterLogin   time.Duration
}

type Config struct {
	URL              string
	Time             string
	TelegramBotToken string
	TelegramChatID   int64
	Interval         time.Duration
	Screen           ScreenConfig
	Login            *LoginConfig
}

var (
	config  Config
	bot     *tgbotapi.BotAPI
	logPath string
)

func main() {
	// Инициализируем логирование
	err := initLogging()
	if err != nil {
		log.Printf("Не удалось инициализировать логирование: %v, использую stdout", err)
	}

	log.Println("=== Запуск screenshot демона ===")

	err = loadConfig("config.ini")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем бота Telegram
	bot, err = tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram бота: %v", err)
	}

	log.Printf("Демон запущен. URL: %s, интервал: %v, разрешение: %dx%d",
		config.URL, config.Interval, config.Screen.Width, config.Screen.Height)

	if config.Login != nil {
		log.Println("Режим с авторизацией включен")
	}

	// Запускаем основной цикл
	run()
}

func initLogging() error {
	// Пробуем разные пути для логов
	possiblePaths := []string{
		"/var/log/screenshot-daemon.log",
		os.Getenv("HOME") + "/screenshot-daemon.log",
		"./screenshot-daemon.log",
	}

	for _, path := range possiblePaths {
		// Проверяем можем ли писать в этот путь
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			logPath = path
			log.SetOutput(file)
			log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

			// Закрываем старый файл если открывали до этого
			if file != nil {
				file.Close()
			}

			log.Printf("Логирование инициализировано: %s", path)
			return nil
		}
	}

	return fmt.Errorf("не удалось инициализировать логирование ни в одном из путей: %v", possiblePaths)
}

func checkAndRotateLog() {
	if logPath == "" {
		return
	}

	// Проверяем размер лог-файла
	info, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Printf("Ошибка проверки размера лога: %v", err)
		return
	}

	// 100 МБ лимит
	const maxSize = 100 * 1024 * 1024
	if info.Size() < maxSize {
		return
	}

	// Ротируем лог
	//backupPath := logPath + ".old"

	// Закрываем текущий лог перед ротацией
	if logPath != "" {
		// Переоткрываем файл в режиме перезаписи
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Ошибка ротации лога: %v", err)
			return
		}
		file.Close()
	}

	log.Printf("Лог-файл ротирован (достигнут лимит 100МБ)")
}

func loadConfig(filename string) error {
	cfg, err := ini.Load(filename)
	if err != nil {
		return fmt.Errorf("не удалось загрузить файл конфигурации: %v", err)
	}

	section := cfg.Section("")

	config.URL = section.Key("url").String()
	if config.URL == "" {
		return fmt.Errorf("url не указан в конфигурации")
	}

	timeStr := section.Key("time").String()
	if timeStr == "" {
		return fmt.Errorf("time не указан в конфигурации")
	}

	config.Interval, err = time.ParseDuration(timeStr)
	if err != nil {
		return fmt.Errorf("ошибка парсинга времени: %v", err)
	}

	config.TelegramBotToken = section.Key("telegram_bot_token").String()
	if config.TelegramBotToken == "" {
		return fmt.Errorf("telegram_bot_token не указан в конфигурации")
	}

	chatIDStr := section.Key("telegram_chat_id").String()
	if chatIDStr == "" {
		return fmt.Errorf("telegram_chat_id не указан в конфигурации")
	}

	config.TelegramChatID, err = strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("ошибка парсинга chat_id: %v", err)
	}

	// Загружаем настройки разрешения экрана
	screenStr := section.Key("screen").String()
	if screenStr != "" {
		screenConfig, err := parseScreenConfig(screenStr)
		if err != nil {
			return fmt.Errorf("ошибка парсинга разрешения экрана: %v", err)
		}
		config.Screen = screenConfig
	} else {
		// Значения по умолчанию
		config.Screen = ScreenConfig{Width: 1920, Height: 1080}
	}

	// Проверяем настройки авторизации
	loginURL := section.Key("login_url").String()
	username := section.Key("login_username").String()
	password := section.Key("login_password").String()

	if loginURL != "" && username != "" && password != "" {
		config.Login = &LoginConfig{
			URL:              loginURL,
			Username:         username,
			Password:         password,
			UsernameSelector: section.Key("login_username_selector").MustString("input[name='username']"),
			PasswordSelector: section.Key("login_password_selector").MustString("input[name='password']"),
			SubmitSelector:   section.Key("login_submit_selector").MustString("button[type='submit']"),
			WaitAfterLogin:   time.Duration(section.Key("login_wait_seconds").MustInt(3)) * time.Second,
		}
	}

	return nil
}

func parseScreenConfig(screenStr string) (ScreenConfig, error) {
	// Удаляем все пробелы
	screenStr = strings.ReplaceAll(screenStr, " ", "")

	// Разделяем по символам x, X, или ×
	parts := strings.FieldsFunc(screenStr, func(r rune) bool {
		return r == 'x' || r == 'X' || r == '×'
	})

	if len(parts) != 2 {
		return ScreenConfig{}, fmt.Errorf("неверный формат разрешения. Ожидается: 1024x768")
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return ScreenConfig{}, fmt.Errorf("неверная ширина: %v", err)
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return ScreenConfig{}, fmt.Errorf("неверная высота: %v", err)
	}

	// Проверяем минимальные значения
	if width < 100 || height < 100 {
		return ScreenConfig{}, fmt.Errorf("разрешение слишком маленькое. Минимум: 100x100")
	}

	// Проверяем максимальные значения (разумный предел)
	if width > 10000 || height > 10000 {
		return ScreenConfig{}, fmt.Errorf("разрешение слишком большое. Максимум: 10000x10000")
	}

	return ScreenConfig{
		Width:  width,
		Height: height,
	}, nil
}

func run() {
	// Выполняем сразу при запуске
	takeAndSendScreenshot()

	// Запускаем периодическое выполнение
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			takeAndSendScreenshot()
		}
	}
}

func takeAndSendScreenshot() {
	// Проверяем и ротируем лог при необходимости
	checkAndRotateLog()

	log.Printf("Создание скриншота для %s (разрешение: %dx%d)",
		config.URL, config.Screen.Width, config.Screen.Height)

	// Создаем временный файл для скриншота
	tmpfile, err := os.CreateTemp("", "screenshot-*.png")
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		return
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Создаем скриншот
	err = takeScreenshot(tmpfile)
	if err != nil {
		log.Printf("Ошибка создания скриншота: %v", err)
		return
	}

	// Отправляем в Telegram
	err = sendToTelegram(tmpfile.Name())
	if err != nil {
		log.Printf("Ошибка отправки в Telegram: %v", err)
		return
	}

	log.Printf("Скриншот успешно отправлен в Telegram")
}

func takeScreenshot(output *os.File) error {
	// Создаем контекст с опциями для headless-режима
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("ignore-certificate-errors", true),
		// Устанавливаем размер окна
		chromedp.WindowSize(config.Screen.Width, config.Screen.Height),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Увеличиваем таймаут для сложных страниц
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Выполняем авторизацию если нужно
	if config.Login != nil {
		err := performLogin(ctx)
		if err != nil {
			return fmt.Errorf("ошибка авторизации: %v", err)
		}
		log.Println("Авторизация успешна")
	}

	// Переходим на целевую страницу и делаем скриншот
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(config.URL),
		chromedp.Sleep(3*time.Second), // Ждем загрузки страницы
		// Дополнительно устанавливаем размер viewport (на всякий случай)
		chromedp.EmulateViewport(int64(config.Screen.Width), int64(config.Screen.Height)),
		chromedp.FullScreenshot(&buf, 90),
	)
	if err != nil {
		return fmt.Errorf("ошибка при создании скриншота: %v", err)
	}

	// Записываем скриншот в файл
	_, err = output.Write(buf)
	return err
}

func performLogin(ctx context.Context) error {
	log.Printf("Выполняем авторизацию на %s", config.Login.URL)

	return chromedp.Run(ctx,
		// Переходим на страницу логина
		chromedp.Navigate(config.Login.URL),
		chromedp.Sleep(2*time.Second),

		// Вводим логин
		chromedp.WaitVisible(config.Login.UsernameSelector, chromedp.BySearch),
		chromedp.SendKeys(config.Login.UsernameSelector, config.Login.Username, chromedp.BySearch),

		// Вводим пароль
		chromedp.WaitVisible(config.Login.PasswordSelector, chromedp.BySearch),
		chromedp.SendKeys(config.Login.PasswordSelector, config.Login.Password, chromedp.BySearch),

		// Нажимаем кнопку входа
		chromedp.WaitVisible(config.Login.SubmitSelector, chromedp.BySearch),
		chromedp.Click(config.Login.SubmitSelector, chromedp.BySearch),

		// Ждем после логина
		chromedp.Sleep(config.Login.WaitAfterLogin),
	)
}

func sendToTelegram(filename string) error {
	// Открываем файл
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Создаем сообщение с фото
	photoConfig := tgbotapi.NewPhoto(config.TelegramChatID, tgbotapi.FileReader{
		Name:   "screenshot.png",
		Reader: file,
	})

	status := "публичная"
	if config.Login != nil {
		status = "защищенная (требовалась авторизация)"
	}

	photoConfig.Caption = fmt.Sprintf("Скриншот %s\nСтатус: %s\nРазрешение: %dx%d\nВремя: %s",
		config.URL, status, config.Screen.Width, config.Screen.Height,
		time.Now().Format("2006-01-02 15:04:05"))

	// Отправляем сообщение
	_, err = bot.Send(photoConfig)
	return err
}
