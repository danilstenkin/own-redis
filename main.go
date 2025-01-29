package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Структура для хранения ключей и TTL
type Storage struct {
	data map[string]string
	ttl  map[string]time.Time
	mu   sync.Mutex
}

func (s *Storage) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	if ttl > 0 {
		s.ttl[key] = time.Now().Add(ttl)
	} else {
		delete(s.ttl, key)
	}
}

func (s *Storage) Get(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, не истёк ли TTL
	if exp, exists := s.ttl[key]; exists {
		if time.Now().After(exp) {
			delete(s.data, key)
			delete(s.ttl, key)
			return "(nil)"
		}
	}
	val, ok := s.data[key]
	if !ok {
		return "(nil)"
	}
	return val
}

func main() {
	port := "8080"

	if len(os.Args) > 2 && os.Args[1] == "--port" {
		port = os.Args[2]
	}

	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		fmt.Println("Ошибка создания UDP-адреса:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		return
	}
	defer conn.Close()

	fmt.Println("UDP-сервер запущен на порту", port)

	// Создаём хранилище
	store := &Storage{
		data: make(map[string]string),
		ttl:  make(map[string]time.Time),
	}

	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Ошибка чтения данных:", err)
			continue
		}

		message := strings.TrimSpace(string(buffer[:n]))
		if message != "" {
			fmt.Println("Получено:", message)
		} else {
			continue
		}

		// Разбираем команду
		args := strings.Fields(message)
		if len(args) == 0 {
			continue
		}

		command := strings.ToUpper(args[0])

		var response string

		switch command {
		case "PING":
			response = "PONG"

		case "SET":
			if len(args) < 3 {
				response = "(error) ERR wrong number of arguments for 'SET' command"
			} else {
				key := args[1]
				value := strings.Join(args[2:], " ")
				var ttl time.Duration

				// Проверяем наличие PX
				if len(args) > 3 && strings.ToUpper(args[len(args)-2]) == "PX" {
					px, err := time.ParseDuration(args[len(args)-1] + "ms")
					if err == nil {
						ttl = px
						value = strings.Join(args[2:len(args)-2], " ") // Исключаем PX
					}
				}

				store.Set(key, value, ttl)
				response = "OK"
			}

		case "GET":
			if len(args) != 2 {
				response = "(error) ERR wrong number of arguments for 'GET' command"
			} else {
				response = store.Get(args[1])
			}

		default:
			response = "(error) ERR unknown command"
		}

		// Отправляем ответ
		_, err = conn.WriteToUDP([]byte(response), clientAddr)
		if err != nil {
			fmt.Println("Ошибка отправки данных:", err)
		}
	}
}
