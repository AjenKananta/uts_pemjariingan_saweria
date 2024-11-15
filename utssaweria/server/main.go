package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Struktur data donasi
type Donation struct {
	Name    string `json:"name"`
	Amount  int    `json:"amount"`
	Message string `json:"message"`
}

// Variabel global
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Izinkan semua origin untuk WebSocket
	},
}
var connections []*websocket.Conn
var mutex sync.Mutex
var balances = make(map[string]int) // Menyimpan saldo per pengguna

func main() {
	// Goroutine untuk menjalankan WebSocket server
	go startWebSocketServer()

	// Goroutine untuk menjalankan UDP server untuk top-up saldo
	go startUDPServer()

	// Goroutine untuk menjalankan TCP server untuk cek saldo
	go startTCPServer()

	// Menjalankan HTTP server untuk website
	http.Handle("/", http.FileServer(http.Dir("../web")))
	fmt.Println("Server berjalan di http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Fungsi untuk memulai WebSocket server
func startWebSocketServer() {
	http.HandleFunc("/ws", handleWebSocket)
	http.ListenAndServe(":8081", nil)
}

// Fungsi untuk menangani koneksi WebSocket
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer conn.Close()

	mutex.Lock()
	connections = append(connections, conn)
	mutex.Unlock()

	// Menerima pesan donasi dari terminal
	for {
		var donation Donation
		if err := conn.ReadJSON(&donation); err != nil {
			fmt.Println("WebSocket error:", err)
			return
		}
		broadcastDonation(donation)
	}
}

// Fungsi untuk broadcast donasi ke semua koneksi
func broadcastDonation(donation Donation) {
	message, _ := json.Marshal(donation)
	mutex.Lock()
	for _, conn := range connections {
		conn.WriteMessage(websocket.TextMessage, message)
	}
	mutex.Unlock()
}

// Fungsi untuk memulai UDP server untuk top-up saldo
func startUDPServer() {
	addr, _ := net.ResolveUDPAddr("udp", ":8082")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	fmt.Println("UDP server untuk top-up saldo berjalan di :8082")

	for {
		buffer := make([]byte, 1024)
		n, clientAddr, _ := conn.ReadFromUDP(buffer)

		var data map[string]interface{}
		json.Unmarshal(buffer[:n], &data)
		name := data["name"].(string)
		amount := int(data["amount"].(float64))

		mutex.Lock()
		balances[name] += amount
		mutex.Unlock()

		fmt.Printf("Top-up saldo %d untuk %s berhasil\n", amount, name)
		conn.WriteToUDP([]byte("Top-up berhasil"), clientAddr)
	}
}

// Fungsi untuk memulai TCP server untuk cek saldo
func startTCPServer() {
	listener, _ := net.Listen("tcp", ":8083")
	defer listener.Close()

	fmt.Println("TCP server untuk cek saldo berjalan di :8083")

	for {
		conn, _ := listener.Accept()
		go handleTCPConnection(conn)
	}
}

// Fungsi untuk menangani koneksi TCP untuk cek saldo
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, _ := conn.Read(buffer)
	name := string(buffer[:n])

	mutex.Lock()
	balance := balances[name]
	mutex.Unlock()

	response := fmt.Sprintf("Saldo untuk %s adalah %d", name, balance)
	conn.Write([]byte(response))
}
