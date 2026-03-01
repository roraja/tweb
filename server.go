package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed static/index.html
var indexHTML []byte

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	validTokens = make(map[string]bool)
	tokenMu     sync.RWMutex
)

func startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/", authMiddleware(handleIndex))
	mux.HandleFunc("/api/terminals", authMiddleware(handleTerminals))
	mux.HandleFunc("/api/terminals/", authMiddleware(handleTerminalAction))
	mux.HandleFunc("/ws/", authMiddleware(handleWS))

	addr := fmt.Sprintf(":%d", config.Port)
	srv := &http.Server{Addr: addr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("Shutting down...")
		closeAllTerminals()
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// Auth

func generateToken() string {
	mac := hmac.New(sha256.New, []byte(serverSecret))
	mac.Write([]byte(fmt.Sprintf("tweb-%d", time.Now().UnixNano())))
	return hex.EncodeToString(mac.Sum(nil))
}

func addToken(token string) {
	tokenMu.Lock()
	validTokens[token] = true
	tokenMu.Unlock()
}

func checkToken(token string) bool {
	tokenMu.RLock()
	defer tokenMu.RUnlock()
	return validTokens[token]
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.Password == "" {
			next(w, r)
			return
		}
		cookie, err := r.Cookie("tweb_session")
		if err != nil || !checkToken(cookie.Value) {
			if r.Header.Get("Upgrade") == "websocket" || strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// Handlers

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, loginHTML(""))
		return
	}
	if r.Method == "POST" {
		r.ParseForm()
		if r.FormValue("password") == config.Password {
			token := generateToken()
			addToken(token)
			http.SetCookie(w, &http.Cookie{
				Name:     "tweb_session",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, loginHTML("Invalid password"))
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(indexHTML)
}

func handleTerminals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listTerminals())
	case "POST":
		info, err := createTerminal()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTerminalAction(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/terminals/")
	if id == "" {
		http.Error(w, "Missing terminal ID", http.StatusBadRequest)
		return
	}
	if r.Method == "DELETE" {
		if err := closeTerminal(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/ws/")
	term := getTerminal(id)
	if term == nil {
		http.Error(w, "Terminal not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// PTY → WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := term.Read(buf)
			if err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket → PTY
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if msgType == websocket.TextMessage {
			var ctrl struct {
				Type string `json:"type"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal(msg, &ctrl) == nil && ctrl.Type == "resize" {
				term.Resize(ctrl.Cols, ctrl.Rows)
			}
			continue
		}
		term.Write(msg)
	}

	<-done
}

// Login page template

func loginHTML(errMsg string) string {
	errorBlock := ""
	if errMsg != "" {
		errorBlock = `<p class="error">` + errMsg + `</p>`
	}
	return `<!DOCTYPE html><html><head><title>tweb - Login</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#1e1e2e;color:#cdd6f4;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;display:flex;align-items:center;justify-content:center;height:100vh}
.login{background:#313244;padding:2rem;border-radius:8px;width:320px}
.login h1{font-size:1.4rem;margin-bottom:1.2rem;color:#89b4fa}
.login input{width:100%;padding:0.6rem;margin-bottom:1rem;background:#1e1e2e;border:1px solid #45475a;color:#cdd6f4;border-radius:4px;font-size:1rem}
.login input:focus{outline:none;border-color:#89b4fa}
.login button{width:100%;padding:0.6rem;background:#89b4fa;color:#1e1e2e;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600}
.login button:hover{background:#74c7ec}
.error{color:#f38ba8;margin-bottom:0.8rem;font-size:0.9rem}
</style></head><body>
<form class="login" method="POST" action="/login">
<h1>🖥 tweb</h1>` + errorBlock + `
<input type="password" name="password" placeholder="Password" autofocus required>
<button type="submit">Login</button>
</form></body></html>`
}
