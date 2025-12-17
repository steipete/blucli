package config

import "testing"

func TestParseDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		host string
		port int
	}{
		{name: "HostOnly", in: "192.168.1.10", host: "192.168.1.10", port: 11000},
		{name: "HostPort", in: "192.168.1.10:12000", host: "192.168.1.10", port: 12000},
		{name: "HTTPURL", in: "http://192.168.1.10:12000", host: "192.168.1.10", port: 12000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, err := ParseDevice(tt.in)
			if err != nil {
				t.Fatalf("ParseDevice() err = %v", err)
			}
			if device.Host != tt.host || device.Port != tt.port {
				t.Fatalf("ParseDevice() = %+v; want host=%q port=%d", device, tt.host, tt.port)
			}
		})
	}
}
