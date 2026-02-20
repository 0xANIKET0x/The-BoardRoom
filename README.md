# 🎨 The BoardRoom: Lightning-Fast Collaborative Whiteboard

[![Go](https://img.shields.io/badge/Go-1.25.0+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Redis](https://img.shields.io/badge/Redis-Upstash-DC382D?style=flat-square&logo=redis)](https://redis.io/)
[![HTML5 Canvas](https://img.shields.io/badge/HTML5-Canvas-E34F26?style=flat-square&logo=html5)](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API)
[![Deployed on Render](https://img.shields.io/badge/Deployed-Render-46E3B7?style=flat-square&logo=render)](https://render.com)

**🚀 Try it live:** [The BoardRoom](https://the-boardroom-t8u3.onrender.com/)

Welcome to **The BoardRoom**! This is a high-performance, real-time collaborative whiteboard and chat application. Designed for speed and seamless teamwork, it allows multiple users to join isolated sessions, draw simultaneously, and chat with zero perceptible latency. 

Whether you are sketching out system architectures, brainstorming with a remote team, or just doodling with friends, The BoardRoom keeps everyone in perfect sync.

## ✨ Why it's Awesome
* **Insta-Sync Collaboration:** Sub-millisecond stroke and chat synchronization powered by Go WebSockets.
* **Private Rooms:** Jump into the "Lobby" or create a custom Room ID for isolated, secure sessions.
* **Silky Smooth 60fps:** Built entirely with Vanilla JavaScript and the HTML5 Canvas API—no heavy frontend frameworks slowing down your brush strokes.
* **Smart Tooling:** Fully equipped with pencils, erasers, shape generators, text injection, bounded area flood-fill, and a magic "Cut & Move" selector.
* **Persistent History:** Late to the meeting? No problem. The board perfectly reconstructs the entire drawing history the second you connect.
* **Export to PDF:** Native client-side rendering lets you download the active canvas as a PDF instantly.

## 🏗 Under the Hood

The BoardRoom is built to handle heavy concurrent connections without dropping frames or losing data.

* **The Engine (Go):** A custom Go WebSocket server handles connection upgrading, room routing, and concurrency locks (`sync.Mutex`), ensuring thread-safe operations.
* **The Nervous System (Redis):** Redis Pub/Sub sits at the center of the app. When a user draws a line, Go publishes the payload to a Redis channel, which instantly broadcasts that stroke to all other active WebSockets in that specific room.
* **The Canvas (Vanilla JS):** A dual-layer canvas architecture (a "draft" layer for active shapes/selections and a "committed" layer for permanent ink) keeps rendering optimized.

## 🧠 Engineering Highlights

Building a real-time app means solving some incredibly fun edge cases. Here are a few cool things happening behind the scenes:

* **Defeating the "Echo" Effect:** In a Pub/Sub model, when you broadcast a stroke, the server sends it back to *everyone*—including you! To prevent "double-drawing" (which causes jagged, thick lines), the frontend injects a randomized, unique cryptographic ID into its session. When the server broadcasts a stroke, the client mathematically filters out its own echoes in O(1) time.
* **Bulletproof JSON Parsing:** Sending nested spatial data (X/Y coordinates, hex colors, tool types) through Go's `json.RawMessage` into Redis can cause double-stringified payloads. The frontend utilizes a custom, recursive `safeJSONParse` wrapper that dynamically unwraps nested layers, guaranteeing stable history reconstruction when new users join.
* **Smart Flood-Fill:** The "Fill" tool doesn't just paint a square; it uses a custom Uint32 Flood Fill algorithm operating directly on the canvas image data buffer to detect pixel boundaries and fill organic shapes instantly.

## 🛠 Run it Locally

Want to mess around with the code or host your own BoardRoom? 

1. **Clone the repository**
   ```bash
   git clone [https://github.com/0xANIKET0x/The-BoardRoom.git](https://github.com/0xANIKET0x/The-BoardRoom.git)
   cd The-BoardRoom
   ```

2. **Set up Redis**
   Ensure you have a local Redis instance running on port `6379`, or set a cloud URL (like Upstash) in your environment variables:
   ```bash
   export REDIS_URL="rediss://your-cloud-redis-url"
   ```

3. **Run the Go Server**
   ```bash
   go run main.go
   ```

4. **Start Drawing**
   Open your browser and navigate to `http://localhost:8080`.

---
*Built with ❤️ by Aniket.*