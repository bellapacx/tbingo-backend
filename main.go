package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Player represents a joined player
type Player struct {
	PhoneNumber string `json:"phoneNumber"`
	CardID      int    `json:"cardId"`
}

var (
	players     = make(map[string]Player) // key: phone number
	playersLock = sync.Mutex{}

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clients     = make(map[*websocket.Conn]bool)
	clientsLock = sync.Mutex{}

	roundStarted  = false
	numberPool    []int
	calledNumbers []int
)

func main() {
	router := gin.Default()

	router.POST("/join", joinHandler)
	router.GET("/ws", wsHandler)

	go broadcaster()

	// Start Telegram bot in background goroutine
	go startTelegramBot()

	log.Println("Starting server on :8080")
	router.Run(":8080")
}

// joinHandler lets a player join a round with phone number and card id
func joinHandler(c *gin.Context) {
	if roundStarted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Round already started"})
		return
	}

	var req struct {
		PhoneNumber string `json:"phoneNumber"`
		CardID      int    `json:"cardId"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	playersLock.Lock()
	defer playersLock.Unlock()

	// Check if this phone number already joined
	if _, exists := players[req.PhoneNumber]; exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already joined"})
		return
	}

	// Check if card already taken
	for _, p := range players {
		if p.CardID == req.CardID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Card already taken"})
			return
		}
	}

	players[req.PhoneNumber] = Player{PhoneNumber: req.PhoneNumber, CardID: req.CardID}
	log.Printf("Player joined: %s with card %d\n", req.PhoneNumber, req.CardID)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Joined successfully",
		"playersCount": len(players),
	})

	if len(players) >= 3 && !roundStarted {
		go startRound()
	}
}

// wsHandler upgrades HTTP to WebSocket and keeps connection alive for pushing updates
func wsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer func() {
		clientsLock.Lock()
		delete(clients, conn)
		clientsLock.Unlock()
		conn.Close()
	}()

	clientsLock.Lock()
	clients[conn] = true
	clientsLock.Unlock()

	// Read loop to keep connection alive (ignore messages)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// broadcaster sends game state every second to all WebSocket clients if round started
func broadcaster() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if roundStarted {
			broadcastGameState()
		}
	}
}

func broadcastGameState() {
	playersLock.Lock()
	msg := gin.H{
		"type":          "update",
		"calledNumbers": calledNumbers,
		"playersCount":  len(players),
		"roundStarted":  roundStarted,
	}
	playersLock.Unlock()

	clientsLock.Lock()
	defer clientsLock.Unlock()

	for client := range clients {
		if err := client.WriteJSON(msg); err != nil {
			log.Println("WebSocket write error:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// startRound begins the bingo round, draws numbers every 3 seconds and broadcasts updates
func startRound() {
	playersLock.Lock()
	roundStarted = true
	numberPool = shuffleNumbers(75)
	calledNumbers = []int{}
	playersLock.Unlock()

	log.Println("Round started!")

	for len(numberPool) > 0 {
		time.Sleep(3 * time.Second)

		playersLock.Lock()
		nextNumber := numberPool[0]
		numberPool = numberPool[1:]
		calledNumbers = append(calledNumbers, nextNumber)
		playersLock.Unlock()

		broadcastGameState()
	}

	log.Println("Round ended!")

	playersLock.Lock()
	roundStarted = false
	players = make(map[string]Player) // reset for next round
	playersLock.Unlock()
}

// shuffleNumbers returns a shuffled slice of integers from 1 to n
func shuffleNumbers(n int) []int {
	nums := make([]int, n)
	for i := 0; i < n; i++ {
		nums[i] = i + 1
	}

	// Simple Fisher-Yates shuffle with time-based seed
	for i := n - 1; i > 0; i-- {
		j := int(time.Now().UnixNano() % int64(i+1))
		nums[i], nums[j] = nums[j], nums[i]
	}
	return nums
}
