package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
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
var mutex sync.Mutex
var balances = make(map[string]int) // Menyimpan saldo per pengguna

func main() {
	// Memulai menu interaktif
	startMenu()
}

// Fungsi untuk menjalankan menu utama
func startMenu() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n--- Menu Saweria ---")
		fmt.Println("1. Top-up saldo")
		fmt.Println("2. Cek saldo")
		fmt.Println("3. Donasi")
		fmt.Println("4. Keluar")
		fmt.Print("Pilih opsi: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			handleTopUp(reader)
		case "2":
			handleCheckBalance(reader)
		case "3":
			handleDonation(reader)
		case "4":
			fmt.Println("Terima kasih telah menggunakan Saweria!")
			return
		default:
			fmt.Println("Pilihan tidak valid, silakan coba lagi.")
		}
	}
}

// Fungsi untuk menangani top-up saldo
func handleTopUp(reader *bufio.Reader) {
	fmt.Print("Masukkan nama: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Masukkan nominal top-up: ")
	amountStr, _ := reader.ReadString('\n')
	amount, err := strconv.Atoi(strings.TrimSpace(amountStr))
	if err != nil {
		fmt.Println("Nominal tidak valid.")
		return
	}

	// Mengirim data top-up melalui UDP
	addr, _ := net.ResolveUDPAddr("udp", "localhost:8082")
	conn, _ := net.DialUDP("udp", nil, addr)
	defer conn.Close()

	data := map[string]interface{}{
		"name":   name,
		"amount": amount,
	}
	jsonData, _ := json.Marshal(data)
	conn.Write(jsonData)

	// Update saldo di peta balances
	mutex.Lock()
	balances[name] += amount
	mutex.Unlock()

	fmt.Println("Top-up berhasil.")
}

// Fungsi untuk menangani cek saldo
func handleCheckBalance(reader *bufio.Reader) {
	fmt.Print("Masukkan nama: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	// Mengirim permintaan cek saldo melalui TCP
	conn, _ := net.Dial("tcp", "localhost:8083")
	defer conn.Close()

	conn.Write([]byte(name))
	buffer := make([]byte, 1024)
	n, _ := conn.Read(buffer)

	fmt.Println("Hasil cek saldo:", string(buffer[:n]))
}

// Fungsi untuk menangani donasi
func handleDonation(reader *bufio.Reader) {
	fmt.Print("Masukkan nama: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Masukkan nominal donasi: ")
	amountStr, _ := reader.ReadString('\n')
	amount, err := strconv.Atoi(strings.TrimSpace(amountStr))
	if err != nil {
		fmt.Println("Nominal tidak valid.")
		return
	}

	fmt.Print("Masukkan pesan donasi: ")
	message, _ := reader.ReadString('\n')
	message = strings.TrimSpace(message)

	// Pengecekan saldo
	mutex.Lock()
	balance := balances[name]
	if balance < amount {
		fmt.Println("Saldo tidak cukup untuk melakukan donasi.")
		mutex.Unlock()
		return
	}
	// Kurangi saldo setelah pengecekan berhasil
	balances[name] -= amount
	mutex.Unlock()

	// Mengirim donasi melalui WebSocket
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8081/ws", nil)
	if err != nil {
		fmt.Println("WebSocket error:", err)
		return
	}
	defer conn.Close()

	donation := Donation{
		Name:    name,
		Amount:  amount,
		Message: message,
	}
	err = conn.WriteJSON(donation)
	if err != nil {
		fmt.Println("Error sending donation:", err)
		return
	}

	fmt.Println("Donasi berhasil terkirim.")
}
