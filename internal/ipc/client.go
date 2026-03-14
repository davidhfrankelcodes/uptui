package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"uptui/internal/models"
)

type Client struct {
	addr string
}

func NewClient(addr string) *Client {
	return &Client{addr: addr}
}

func (c *Client) do(req Request) (Response, error) {
	conn, err := net.DialTimeout("tcp", c.addr, 3*time.Second)
	if err != nil {
		return Response{}, fmt.Errorf("cannot connect to daemon at %s: %w", c.addr, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, fmt.Errorf("send: %w", err)
	}

	var resp Response
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return Response{}, fmt.Errorf("decode response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return Response{}, fmt.Errorf("read: %w", err)
	}
	if !resp.OK {
		return resp, fmt.Errorf("daemon: %s", resp.Error)
	}
	return resp, nil
}

func (c *Client) Ping() bool {
	conn, err := net.DialTimeout("tcp", c.addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (c *Client) List() ([]*models.MonitorStatus, error) {
	resp, err := c.do(Request{Action: ActionList})
	if err != nil {
		return nil, err
	}
	return resp.Monitors, nil
}

func (c *Client) Add(m models.Monitor) (*models.MonitorStatus, error) {
	resp, err := c.do(Request{Action: ActionAdd, Monitor: &m})
	if err != nil {
		return nil, err
	}
	return resp.Monitor, nil
}

func (c *Client) Delete(name string) error {
	_, err := c.do(Request{Action: ActionDelete, Name: name})
	return err
}

func (c *Client) Pause(name string) error {
	_, err := c.do(Request{Action: ActionPause, Name: name})
	return err
}

func (c *Client) Resume(name string) error {
	_, err := c.do(Request{Action: ActionResume, Name: name})
	return err
}

func (c *Client) Edit(oldName string, m models.Monitor) (*models.MonitorStatus, error) {
	resp, err := c.do(Request{Action: ActionEdit, OldName: oldName, Monitor: &m})
	if err != nil {
		return nil, err
	}
	return resp.Monitor, nil
}

func (c *Client) Reload() error {
	_, err := c.do(Request{Action: ActionReload})
	return err
}
