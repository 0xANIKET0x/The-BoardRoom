package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client
var addr = flag.String("addr", ":8080", "http service address")

var upgrader = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }

type Message struct {
	Type     string          `json:"type"`     
	Room     string          `json:"room"`     
	Username string          `json:"username"` 
	Payload  json.RawMessage `json:"payload"`  
}

type ClientManager struct {
	rooms map[string]map[*websocket.Conn]bool
	mutex sync.Mutex
}

var manager = ClientManager{ rooms: make(map[string]map[*websocket.Conn]bool) }

func initRedis() {
	// Look for Docker environment variable, fallback to localhost if running directly
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	if _, err := rdb.Ping(ctx).Result(); err != nil { log.Fatal("Redis Error:", err) }
	log.Println("✅ Redis connected to:", redisAddr)
}


func subscribeToRedis() {
	pubsub := rdb.Subscribe(ctx, "live_updates")
	defer pubsub.Close()
	ch := pubsub.Channel()
	for msg := range ch {
		var parsedMsg Message
		json.Unmarshal([]byte(msg.Payload), &parsedMsg)
		manager.mutex.Lock()
		if clients, ok := manager.rooms[parsedMsg.Room]; ok {
			for client := range clients {
				client.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			}
		}
		manager.mutex.Unlock()
	}
}

// --- ADMIN API ---
func handleAdminList(w http.ResponseWriter, r *http.Request) {
	rooms, _ := rdb.SMembers(ctx, "active_rooms").Result()
	json.NewEncoder(w).Encode(rooms)
}

func handleAdminDestroy(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	rdb.Del(ctx, "history:"+room)
	rdb.SRem(ctx, "active_rooms", room)
	
	// Force Clear
	clearMsg := Message{Type: "clear", Room: room, Payload: nil}
	bytes, _ := json.Marshal(clearMsg)
	rdb.Publish(ctx, "live_updates", bytes)
	
	w.Write([]byte("Destroyed"))
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	user := r.URL.Query().Get("user")
	if room == "" { room = "general" }
	if user == "" { user = "Anon" }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil { log.Fatal(err) }
	defer ws.Close()

	rdb.SAdd(ctx, "active_rooms", room)

	manager.mutex.Lock()
	if _, ok := manager.rooms[room]; !ok { manager.rooms[room] = make(map[*websocket.Conn]bool) }
	manager.rooms[room][ws] = true
	manager.mutex.Unlock()

	// HISTORY LOAD
	historyKey := "history:" + room
	rawHistory, _ := rdb.LRange(ctx, historyKey, 0, -1).Result()
	
	cleanHistory := make([]json.RawMessage, 0)
	for _, s := range rawHistory {
		cleanHistory = append(cleanHistory, json.RawMessage(s))
	}

	if len(cleanHistory) > 0 {
		payloadBytes, _ := json.Marshal(cleanHistory)
		loadMsg := Message{Type: "history_load", Room: room, Payload: payloadBytes}
		finalBytes, _ := json.Marshal(loadMsg)
		ws.WriteMessage(websocket.TextMessage, finalBytes)
	}

	for {
		_, msgData, err := ws.ReadMessage()
		if err != nil {
			manager.mutex.Lock(); delete(manager.rooms[room], ws); manager.mutex.Unlock(); break
		}

		var msg Message
		json.Unmarshal(msgData, &msg)
		msg.Room = room; msg.Username = user
		finalMsg, _ := json.Marshal(msg)

		if msg.Type == "clear" {
			rdb.Del(ctx, historyKey)
			rdb.Publish(ctx, "live_updates", finalMsg)
		} else if msg.Type == "undo" {
			// Skip Chat Messages logic
			poppedChats := make([]string, 0)
			for {
				val, err := rdb.RPop(ctx, historyKey).Result()
				if err != nil { break }
				var lastMsg Message
				json.Unmarshal([]byte(val), &lastMsg)
				if lastMsg.Type == "chat" { poppedChats = append(poppedChats, val) } else { break }
			}
			for i := len(poppedChats) - 1; i >= 0; i-- { rdb.RPush(ctx, historyKey, poppedChats[i]) }

			// Refresh
			newRaw, _ := rdb.LRange(ctx, historyKey, 0, -1).Result()
			newClean := make([]json.RawMessage, 0)
			for _, s := range newRaw { newClean = append(newClean, json.RawMessage(s)) }
			payloadBytes, _ := json.Marshal(newClean)
			
			refreshMsg := Message{Type: "refresh", Room: room, Payload: payloadBytes}
			finalRefresh, _ := json.Marshal(refreshMsg)
			rdb.Publish(ctx, "live_updates", finalRefresh)
		} else {
			rdb.RPush(ctx, historyKey, finalMsg)
			rdb.Publish(ctx, "live_updates", finalMsg)
		}
	}
}

func main() {
	flag.Parse(); initRedis(); go subscribeToRedis()
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/admin/rooms", handleAdminList)
	http.HandleFunc("/admin/destroy", handleAdminDestroy)
	log.Printf("🚀 Server started on %s", *addr)
	http.ListenAndServe(*addr, nil)
}
