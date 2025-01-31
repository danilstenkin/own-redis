package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	conn    net.PacketConn
	storage map[string]Value
	mu      sync.Mutex
}

type Value struct {
	data      string
	expiresAt time.Time
}

func NewServer(port string) (*Server, error) {
	conn, err := net.ListenPacket("udp", ":"+port)
	if err != nil {
		return nil, err
	}

	return &Server{
		conn:    conn,
		storage: make(map[string]Value),
	}, nil
}

func (s *Server) HandleRequest() {
	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := s.conn.ReadFrom(buffer)
		if err != nil {
			fmt.Println("Error reading:", err)
			continue
		}

		request := strings.TrimSpace(string(buffer[:n]))
		fmt.Println("Received from", clientAddr, ":", request)

		response := s.proccessComand(request)

		_, err = s.conn.WriteTo([]byte(response), clientAddr)
		if err != nil {
			fmt.Println("Error sending response:", err)
		}
	}
}

func (s *Server) proccessComand(request string) string {
	args := strings.Split(request, " ")

	if len(args) == 0 {
		return "(error) empty command"
	}

	switch strings.ToLower(args[0]) {
	case "ping":
		return "PONG\n"
	case "set":
		return s.handleSet(args)
	case "get":
		return s.handleGet(args)
	default:
		return "(error) invalid command\n"
	}
}

func (s *Server) handleSet(args []string) string {
	if len(args) < 3 {
		return "(error) ERR wrong number of arguments for 'SET' command\n"
	}

	key := args[1]
	value := ""
	var ttl time.Duration

	for i := 2; i < len(args); i++ {
		if strings.ToUpper(args[i]) == "PX" {
			if i+1 >= len(args) {
				return "(error) ERR syntax error in PX"
			}

			pxValue, err := time.ParseDuration(args[i+1] + "ms")
			if err != nil || pxValue <= 0 {
				return "(error) ERR invalid PX value"
			}
			ttl = pxValue
			break
		}
		value += args[i] + " "
	}

	value = strings.TrimSpace(value)

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	s.mu.Lock()
	s.storage[key] = Value{data: value + "\n", expiresAt: expiresAt}
	s.mu.Unlock()

	return "OK\n"
}

func (s *Server) handleGet(args []string) string {
	if len(args) != 2 {
		return "(error) ERR wrong number of arguments for 'GET' command"
	}

	key := args[1]

	s.mu.Lock()
	val, exists := s.storage[key]

	if !exists || (!val.expiresAt.IsZero() && time.Now().After(val.expiresAt)) {
		delete(s.storage, key)
		s.mu.Unlock()
		return "(nil)\n"
	}

	s.mu.Unlock()
	return val.data
}
