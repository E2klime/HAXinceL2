package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/E2klime/HAXinceL2/internal/protocol"
)

func (c *Client) handleFileRead(msg *protocol.Message) {
	var payload protocol.FileReadPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse file read payload", err)
		return
	}

	log.Printf("Reading file: %s", payload.Path)

	data, err := os.ReadFile(payload.Path)
	if err != nil {
		c.sendError(fmt.Sprintf("Failed to read file %s", payload.Path), err)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	c.sendResponse(true, encoded, "")

	log.Printf("File read successfully: %s (%d bytes)", payload.Path, len(data))
}

func (c *Client) handleFileWrite(msg *protocol.Message) {
	var payload protocol.FileWritePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse file write payload", err)
		return
	}

	log.Printf("Writing file: %s", payload.Path)

	data, err := base64.StdEncoding.DecodeString(payload.Content)
	if err != nil {
		c.sendError("Failed to decode file content", err)
		return
	}

	mode := fs.FileMode(0644)
	if payload.Mode != 0 {
		mode = fs.FileMode(payload.Mode)
	}

	dir := filepath.Dir(payload.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.sendError("Failed to create directory", err)
		return
	}

	if err := os.WriteFile(payload.Path, data, mode); err != nil {
		c.sendError(fmt.Sprintf("Failed to write file %s", payload.Path), err)
		return
	}

	c.sendResponse(true, fmt.Sprintf("File written successfully: %d bytes", len(data)), "")
	log.Printf("File written successfully: %s (%d bytes)", payload.Path, len(data))
}

func (c *Client) handleFileDelete(msg *protocol.Message) {
	var payload protocol.FileDeletePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse file delete payload", err)
		return
	}

	log.Printf("Deleting file/directory: %s", payload.Path)

	info, err := os.Stat(payload.Path)
	if err != nil {
		c.sendError(fmt.Sprintf("File/directory not found: %s", payload.Path), err)
		return
	}

	if info.IsDir() {
		err = os.RemoveAll(payload.Path)
	} else {
		err = os.Remove(payload.Path)
	}

	if err != nil {
		c.sendError(fmt.Sprintf("Failed to delete %s", payload.Path), err)
		return
	}

	c.sendResponse(true, fmt.Sprintf("Deleted successfully: %s", payload.Path), "")
	log.Printf("Deleted successfully: %s", payload.Path)
}

func (c *Client) handleFileList(msg *protocol.Message) {
	var payload protocol.FileListPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse file list payload", err)
		return
	}

	log.Printf("Listing directory: %s", payload.Path)

	entries, err := os.ReadDir(payload.Path)
	if err != nil {
		c.sendError(fmt.Sprintf("Failed to read directory %s", payload.Path), err)
		return
	}

	var files []protocol.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("Warning: failed to get info for %s: %v", entry.Name(), err)
			continue
		}

		fileInfo := protocol.FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(payload.Path, entry.Name()),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime().Unix(),
			Mode:    info.Mode().String(),
		}
		files = append(files, fileInfo)
	}

	jsonData, err := json.Marshal(files)
	if err != nil {
		c.sendError("Failed to serialize file list", err)
		return
	}

	c.sendResponse(true, string(jsonData), "")
	log.Printf("Directory listed successfully: %s (%d files)", payload.Path, len(files))
}

func (c *Client) handleFileDownload(msg *protocol.Message) {
	var payload protocol.FileDownloadPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse file download payload", err)
		return
	}

	log.Printf("Downloading file: %s", payload.Path)

	info, err := os.Stat(payload.Path)
	if err != nil {
		c.sendError(fmt.Sprintf("File not found: %s", payload.Path), err)
		return
	}

	if info.IsDir() {
		c.sendError("Cannot download directory", fmt.Errorf("path is a directory"))
		return
	}

	data, err := os.ReadFile(payload.Path)
	if err != nil {
		c.sendError(fmt.Sprintf("Failed to read file %s", payload.Path), err)
		return
	}

	result := map[string]interface{}{
		"name":     filepath.Base(payload.Path),
		"path":     payload.Path,
		"size":     info.Size(),
		"mod_time": info.ModTime().Unix(),
		"mode":     info.Mode().String(),
		"content":  base64.StdEncoding.EncodeToString(data),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		c.sendError("Failed to serialize download result", err)
		return
	}

	c.sendResponse(true, string(jsonData), "")
	log.Printf("File downloaded successfully: %s (%d bytes)", payload.Path, len(data))
}
