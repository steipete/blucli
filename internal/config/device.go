package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Device struct {
	ID   string `json:"id"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (d Device) BaseURL() *url.URL {
	host := d.Host
	if host == "" {
		host = "127.0.0.1"
	}

	port := d.Port
	if port == 0 {
		port = 11000
	}

	u := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   "/",
	}
	return u
}

func ParseDevice(s string) (Device, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Device{}, fmt.Errorf("empty device")
	}

	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		u, err := url.Parse(s)
		if err != nil {
			return Device{}, err
		}
		host, portStr, err := net.SplitHostPort(u.Host)
		if err != nil {
			host = u.Host
			portStr = ""
		}
		port := 11000
		if portStr != "" {
			p, err := strconv.Atoi(portStr)
			if err != nil {
				return Device{}, err
			}
			port = p
		}
		id := fmt.Sprintf("%s:%d", host, port)
		return Device{ID: id, Host: host, Port: port}, nil
	}

	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		host = s
		portStr = ""
	}

	port := 11000
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return Device{}, err
		}
		port = p
	}

	host = strings.Trim(host, "[]")
	if host == "" {
		return Device{}, fmt.Errorf("missing host")
	}

	id := fmt.Sprintf("%s:%d", host, port)
	return Device{ID: id, Host: host, Port: port}, nil
}
