//go:build windows

package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/E2klime/HAXinceL2/internal/protocol"
	"golang.org/x/sys/windows/registry"
)

func parseRegistryKey(keyPath string) (registry.Key, string, error) {
	parts := strings.SplitN(keyPath, "\\", 2)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid registry key format: %s", keyPath)
	}

	var rootKey registry.Key
	switch strings.ToUpper(parts[0]) {
	case "HKEY_CLASSES_ROOT", "HKCR":
		rootKey = registry.CLASSES_ROOT
	case "HKEY_CURRENT_USER", "HKCU":
		rootKey = registry.CURRENT_USER
	case "HKEY_LOCAL_MACHINE", "HKLM":
		rootKey = registry.LOCAL_MACHINE
	case "HKEY_USERS", "HKU":
		rootKey = registry.USERS
	case "HKEY_CURRENT_CONFIG", "HKCC":
		rootKey = registry.CURRENT_CONFIG
	default:
		return 0, "", fmt.Errorf("unknown root key: %s", parts[0])
	}

	return rootKey, parts[1], nil
}

func (c *Client) handleRegRead(msg *protocol.Message) {
	var payload protocol.RegistryReadPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse registry read payload", err)
		return
	}

	log.Printf("Reading registry: %s\\%s", payload.Key, payload.Value)

	rootKey, subKey, err := parseRegistryKey(payload.Key)
	if err != nil {
		c.sendError("Invalid registry key", err)
		return
	}

	k, err := registry.OpenKey(rootKey, subKey, registry.QUERY_VALUE)
	if err != nil {
		c.sendError(fmt.Sprintf("Failed to open registry key %s", payload.Key), err)
		return
	}
	defer k.Close()

	value, valType, err := k.GetStringValue(payload.Value)
	if err != nil {
		dwordVal, _, errDword := k.GetIntegerValue(payload.Value)
		if errDword == nil {
			result := map[string]interface{}{
				"key":       payload.Key,
				"value":     payload.Value,
				"data":      fmt.Sprintf("%d", dwordVal),
				"data_type": "dword",
			}
			jsonData, _ := json.Marshal(result)
			c.sendResponse(true, string(jsonData), "")
			return
		}

		binaryVal, _, errBinary := k.GetBinaryValue(payload.Value)
		if errBinary == nil {
			result := map[string]interface{}{
				"key":       payload.Key,
				"value":     payload.Value,
				"data":      fmt.Sprintf("%x", binaryVal),
				"data_type": "binary",
			}
			jsonData, _ := json.Marshal(result)
			c.sendResponse(true, string(jsonData), "")
			return
		}

		c.sendError(fmt.Sprintf("Failed to read registry value %s", payload.Value), err)
		return
	}

	var dataType string
	switch valType {
	case registry.SZ:
		dataType = "string"
	case registry.EXPAND_SZ:
		dataType = "expand_string"
	case registry.MULTI_SZ:
		dataType = "multi_string"
	default:
		dataType = "unknown"
	}

	result := map[string]interface{}{
		"key":       payload.Key,
		"value":     payload.Value,
		"data":      value,
		"data_type": dataType,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		c.sendError("Failed to serialize registry data", err)
		return
	}

	c.sendResponse(true, string(jsonData), "")
	log.Printf("Registry value read successfully: %s\\%s", payload.Key, payload.Value)
}

func (c *Client) handleRegWrite(msg *protocol.Message) {
	var payload protocol.RegistryWritePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse registry write payload", err)
		return
	}

	log.Printf("Writing registry: %s\\%s = %s (%s)", payload.Key, payload.Value, payload.Data, payload.DataType)

	rootKey, subKey, err := parseRegistryKey(payload.Key)
	if err != nil {
		c.sendError("Invalid registry key", err)
		return
	}

	k, _, err := registry.CreateKey(rootKey, subKey, registry.SET_VALUE)
	if err != nil {
		c.sendError(fmt.Sprintf("Failed to create/open registry key %s", payload.Key), err)
		return
	}
	defer k.Close()

	switch strings.ToLower(payload.DataType) {
	case "string", "sz":
		err = k.SetStringValue(payload.Value, payload.Data)
	case "expand_string", "expand_sz":
		err = k.SetExpandStringValue(payload.Value, payload.Data)
	case "dword":
		val, parseErr := strconv.ParseUint(payload.Data, 0, 32)
		if parseErr != nil {
			c.sendError("Invalid DWORD value", parseErr)
			return
		}
		err = k.SetDWordValue(payload.Value, uint32(val))
	case "qword":
		val, parseErr := strconv.ParseUint(payload.Data, 0, 64)
		if parseErr != nil {
			c.sendError("Invalid QWORD value", parseErr)
			return
		}
		err = k.SetQWordValue(payload.Value, val)
	default:
		c.sendError("Unsupported data type", fmt.Errorf("type: %s", payload.DataType))
		return
	}

	if err != nil {
		c.sendError(fmt.Sprintf("Failed to write registry value %s", payload.Value), err)
		return
	}

	c.sendResponse(true, fmt.Sprintf("Registry value written successfully: %s\\%s", payload.Key, payload.Value), "")
	log.Printf("Registry value written successfully: %s\\%s", payload.Key, payload.Value)
}

func (c *Client) handleRegDelete(msg *protocol.Message) {
	var payload protocol.RegistryDeletePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse registry delete payload", err)
		return
	}

	log.Printf("Deleting registry: %s\\%s", payload.Key, payload.Value)

	rootKey, subKey, err := parseRegistryKey(payload.Key)
	if err != nil {
		c.sendError("Invalid registry key", err)
		return
	}

	if payload.Value == "" {
		err = registry.DeleteKey(rootKey, subKey)
		if err != nil {
			c.sendError(fmt.Sprintf("Failed to delete registry key %s", payload.Key), err)
			return
		}
		c.sendResponse(true, fmt.Sprintf("Registry key deleted successfully: %s", payload.Key), "")
		log.Printf("Registry key deleted successfully: %s", payload.Key)
	} else {
		k, err := registry.OpenKey(rootKey, subKey, registry.SET_VALUE)
		if err != nil {
			c.sendError(fmt.Sprintf("Failed to open registry key %s", payload.Key), err)
			return
		}
		defer k.Close()

		err = k.DeleteValue(payload.Value)
		if err != nil {
			c.sendError(fmt.Sprintf("Failed to delete registry value %s", payload.Value), err)
			return
		}

		c.sendResponse(true, fmt.Sprintf("Registry value deleted successfully: %s\\%s", payload.Key, payload.Value), "")
		log.Printf("Registry value deleted successfully: %s\\%s", payload.Key, payload.Value)
	}
}

func (c *Client) handleRegList(msg *protocol.Message) {
	var payload protocol.RegistryListPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Failed to parse registry list payload", err)
		return
	}

	log.Printf("Listing registry: %s", payload.Key)

	rootKey, subKey, err := parseRegistryKey(payload.Key)
	if err != nil {
		c.sendError("Invalid registry key", err)
		return
	}

	k, err := registry.OpenKey(rootKey, subKey, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		c.sendError(fmt.Sprintf("Failed to open registry key %s", payload.Key), err)
		return
	}
	defer k.Close()

	subKeys, err := k.ReadSubKeyNames(-1)
	if err != nil && err.Error() != "EOF" {
		log.Printf("Warning: failed to read subkeys: %v", err)
		subKeys = []string{}
	}

	valueNames, err := k.ReadValueNames(-1)
	if err != nil && err.Error() != "EOF" {
		log.Printf("Warning: failed to read value names: %v", err)
		valueNames = []string{}
	}

	var items []protocol.RegistryInfo

	for _, name := range subKeys {
		items = append(items, protocol.RegistryInfo{
			Name:     name,
			Type:     "key",
			Value:    "",
			DataType: "",
		})
	}

	for _, name := range valueNames {
		value, valType, err := k.GetStringValue(name)
		if err != nil {
			dwordVal, _, errDword := k.GetIntegerValue(name)
			if errDword == nil {
				items = append(items, protocol.RegistryInfo{
					Name:     name,
					Type:     "value",
					Value:    fmt.Sprintf("%d", dwordVal),
					DataType: "dword",
				})
				continue
			}
			binaryVal, _, errBinary := k.GetBinaryValue(name)
			if errBinary == nil {
				items = append(items, protocol.RegistryInfo{
					Name:     name,
					Type:     "value",
					Value:    fmt.Sprintf("%x", binaryVal),
					DataType: "binary",
				})
				continue
			}
			continue
		}

		var dataType string
		switch valType {
		case registry.SZ:
			dataType = "string"
		case registry.EXPAND_SZ:
			dataType = "expand_string"
		case registry.MULTI_SZ:
			dataType = "multi_string"
		default:
			dataType = "unknown"
		}

		items = append(items, protocol.RegistryInfo{
			Name:     name,
			Type:     "value",
			Value:    value,
			DataType: dataType,
		})
	}

	jsonData, err := json.Marshal(items)
	if err != nil {
		c.sendError("Failed to serialize registry list", err)
		return
	}

	c.sendResponse(true, string(jsonData), "")
	log.Printf("Registry listed successfully: %s (%d items)", payload.Key, len(items))
}
