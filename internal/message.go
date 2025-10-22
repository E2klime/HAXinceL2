package internal

import "encoding/json"

type MessageType string

const (
	TypeAuth         MessageType = "auth"
	TypeHeartbeat    MessageType = "heartbeat"
	TypeCommand      MessageType = "command"
	TypeScreenshot   MessageType = "screenshot"
	TypeWebcam       MessageType = "webcam"
	TypeShowImage    MessageType = "show_image"
	TypeResponse     MessageType = "response"
	TypeError        MessageType = "error"
	TypeFileRead     MessageType = "file_read"
	TypeFileWrite    MessageType = "file_write"
	TypeFileDelete   MessageType = "file_delete"
	TypeFileList     MessageType = "file_list"
	TypeFileDownload MessageType = "file_download"
	TypeRegRead      MessageType = "reg_read"
	TypeRegWrite     MessageType = "reg_write"
	TypeRegDelete    MessageType = "reg_delete"
	TypeRegList      MessageType = "reg_list"
)

type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

type AuthPayload struct {
	ClientID string `json:"client_id"`
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	OS       string `json:"os"`
}

type CommandPayload struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type ScreenshotPayload struct {
	Quality int `json:"quality"`
}

type WebcamPayload struct {
	Duration  int    `json:"duration"`
	StreamURL string `json:"stream_url,omitempty"`
}

type ShowImagePayload struct {
	ImageURL string `json:"image_url"`
	Duration int    `json:"duration"`
}

type ResponsePayload struct {
	Success bool   `json:"success"`
	Data    string `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type FileReadPayload struct {
	Path string `json:"path"`
}

type FileWritePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    uint32 `json:"mode,omitempty"`
}

type FileDeletePayload struct {
	Path string `json:"path"`
}

type FileListPayload struct {
	Path string `json:"path"`
}

type FileDownloadPayload struct {
	Path string `json:"path"`
}

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime int64  `json:"mod_time"`
	Mode    string `json:"mode"`
}

type RegistryReadPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RegistryWritePayload struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	DataType string `json:"data_type"`
}

type RegistryDeletePayload struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type RegistryListPayload struct {
	Key string `json:"key"`
}

type RegistryInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	DataType string `json:"data_type"`
}
