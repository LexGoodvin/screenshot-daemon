package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/go-ini/ini"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type LoginConfig struct {
	URL      string
	Username string
	Password string
	// Селекторы для полей ввода (CSS или XPath)
	UsernameSelector string
	PasswordSelector string
	SubmitSelector   string
	// Дополнительные настройки
	WaitAfterLogin time.Duration
}

type Config struct {
	URL              string
	Time             string
	TelegramBotToken string
	TelegramChatID   int64
	Interval         time.Duration
	Login            *LoginConfig
}

var (
	config Config
	bot    *tgbotapi.BotAPI
)

func main() {
	// Настраиваем логгирование
	logFile, err := os.OpenFile("/var/log/screenshot-daemon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Не удалось открыть файл лога: %v, использую stdout", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
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

	log.Printf("Демон запущен. URL: %s, интервал: %v", config.URL, config.Interval)
	if config.Login != nil {
		log.Println("Режим с авторизацией включен")
	}

	// Запускаем основной цикл
	run()
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
	log.Printf("Создание скриншота для %s", config.URL)

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
		chromedp.Flag("ignore-certificate-errors", true), // Игнорируем SSL ошибки
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

	photoConfig.Caption = fmt.Sprintf("Скриншот %s\nСтатус: %s\nВремя: %s",
		config.URL, status, time.Now().Format("2006-01-02 15:04:05"))

	// Отправляем сообщение
	_, err = bot.Send(photoConfig)
	return err
}
