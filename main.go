package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/go-ini/ini"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Config struct {
	URL              string
	Time             string
	TelegramBotToken string
	TelegramChatID   int64
	Interval         time.Duration
}

type ScreenshotDaemon struct {
	config *Config
	bot    *tgbotapi.BotAPI
}

func main() {
	// Загружаем конфигурацию
	config, err := loadConfig("config.ini")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем бота Telegram
	bot, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram бота: %v", err)
	}

	daemon := &ScreenshotDaemon{
		config: config,
		bot:    bot,
	}

	log.Printf("Демон запущен. URL: %s, интервал: %v", config.URL, config.Interval)

	// Запускаем основной цикл
	daemon.run()
}

func loadConfig(filename string) (*Config, error) {
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить файл конфигурации: %v", err)
	}

	section := cfg.Section("")

	url := section.Key("url").String()
	if url == "" {
		return nil, fmt.Errorf("url не указан в конфигурации")
	}

	timeStr := section.Key("time").String()
	if timeStr == "" {
		return nil, fmt.Errorf("time не указан в конфигурации")
	}

	interval, err := parseTime(timeStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга времени: %v", err)
	}

	botToken := section.Key("telegram_bot_token").String()
	if botToken == "" {
		return nil, fmt.Errorf("telegram_bot_token не указан в конфигурации")
	}

	chatIDStr := section.Key("telegram_chat_id").String()
	if chatIDStr == "" {
		return nil, fmt.Errorf("telegram_chat_id не указан в конфигурации")
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга chat_id: %v", err)
	}

	return &Config{
		URL:              url,
		Time:             timeStr,
		TelegramBotToken: botToken,
		TelegramChatID:   chatID,
		Interval:         interval,
	}, nil
}

func parseTime(timeStr string) (time.Duration, error) {
	// Парсим время в формате "1h", "30m", "2h30m", etc.
	return time.ParseDuration(timeStr)
}

func (d *ScreenshotDaemon) run() {
	// Выполняем сразу при запуске
	d.takeAndSendScreenshot()

	// Запускаем периодическое выполнение
	ticker := time.NewTicker(d.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.takeAndSendScreenshot()
		}
	}
}

func (d *ScreenshotDaemon) takeAndSendScreenshot() {
	log.Printf("Создание скриншота для %s", d.config.URL)

	// Создаем временный файл для скриншота
	tmpfile, err := os.CreateTemp("", "screenshot-*.png")
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		return
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Создаем скриншот
	err = d.takeScreenshot(tmpfile)
	if err != nil {
		log.Printf("Ошибка создания скриншота: %v", err)
		return
	}

	// Отправляем в Telegram
	err = d.sendToTelegram(tmpfile.Name())
	if err != nil {
		log.Printf("Ошибка отправки в Telegram: %v", err)
		return
	}

	log.Printf("Скриншот успешно отправлен в Telegram")
}

func (d *ScreenshotDaemon) takeScreenshot(output io.Writer) error {
	// Создаем контекст
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Выполняем действия в браузере
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(d.config.URL),
		chromedp.Sleep(2*time.Second), // Ждем загрузки страницы
		chromedp.FullScreenshot(&buf, 90),
	)
	if err != nil {
		return fmt.Errorf("ошибка при создании скриншота: %v", err)
	}

	// Записываем скриншот в файл
	_, err = output.Write(buf)
	return err
}

func (d *ScreenshotDaemon) sendToTelegram(filename string) error {
	// Открываем файл
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Создаем сообщение с фото
	photoConfig := tgbotapi.NewPhoto(d.config.TelegramChatID, tgbotapi.FileReader{
		Name:   "screenshot.png",
		Reader: file,
	})
	photoConfig.Caption = fmt.Sprintf("Скриншот %s\nВремя: %s", d.config.URL, time.Now().Format("2006-01-02 15:04:05"))

	// Отправляем сообщение
	_, err = d.bot.Send(photoConfig)
	return err
}
