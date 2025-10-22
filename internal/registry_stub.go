//go:build !windows

package internal

import (
	"fmt"

	"github.com/E2klime/HAXinceL2/internal/protocol"
)

func (c *Client) handleRegRead(msg *protocol.Message) {
	c.sendError("Registry operations are only supported on Windows", fmt.Errorf("unsupported platform"))
}

func (c *Client) handleRegWrite(msg *protocol.Message) {
	c.sendError("Registry operations are only supported on Windows", fmt.Errorf("unsupported platform"))
}

func (c *Client) handleRegDelete(msg *protocol.Message) {
	c.sendError("Registry operations are only supported on Windows", fmt.Errorf("unsupported platform"))
}

func (c *Client) handleRegList(msg *protocol.Message) {
	c.sendError("Registry operations are only supported on Windows", fmt.Errorf("unsupported platform"))
}
