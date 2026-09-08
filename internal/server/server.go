package server

import (
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jsandas/bedrock-server/internal/runner"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 15 * time.Second
	webSocketBufSize  = 1024
	outputLimit       = 1000
)

// Server handles the HTTP endpoints and web UI.
type Server struct {
	runner         *runner.Runner
	connections    map[*websocket.Conn]bool
	connLock       sync.RWMutex
	outputBuffer   []string
	authKey        string   // Pre-shared key for authentication
	allowedOrigins []string // Allowed websocket origins.
}

// Config holds configuration for the server.
type Config struct {
	Runner       *runner.Runner
	AuthKey      string
	AllowedHosts []string
}

// New creates a new Server instance.
func New(config Config) *Server {
	if len(config.AllowedHosts) == 0 {
		config.AllowedHosts = []string{"localhost", "127.0.0.1", "::1"}
	}

	srv := &Server{
		runner:         config.Runner,
		connections:    make(map[*websocket.Conn]bool),
		authKey:        config.AuthKey,
		allowedOrigins: config.AllowedHosts,
	}

	// Start goroutine to handle runner output
	go srv.handleRunnerOutput()

	return srv
}

// Start begins the HTTP server.
func (s *Server) Start(addr string) error {
	// Create a new ServeMux for our routes.
	mux := http.NewServeMux()

	// Index page doesn't require auth.
	mux.HandleFunc("/", s.handleIndex)

	// Protected routes with auth middleware.
	mux.HandleFunc("/ws", s.authMiddleware(s.handleWebSocket))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	slog.Default().Info("Web server started", "addr", addr)
	return server.ListenAndServe()
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  webSocketBufSize,
		WriteBufferSize: webSocketBufSize,
		CheckOrigin: func(r *http.Request) bool {
			return isAllowedOrigin(r, s.allowedOriginsList())
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Default().WarnContext(r.Context(), "Error upgrading to WebSocket", "err", err)
		return
	}
	defer conn.Close()

	// Register connection
	s.connLock.Lock()
	s.connections[conn] = true
	s.connLock.Unlock()

	// Clean up on disconnect
	defer func() {
		s.connLock.Lock()
		delete(s.connections, conn)
		s.connLock.Unlock()
	}()

	// Send initial buffer
	s.connLock.RLock()
	for _, line := range s.outputBuffer {
		if writeErr := conn.WriteMessage(websocket.TextMessage, []byte(line)); writeErr != nil {
			s.connLock.RUnlock()
			return
		}
	}
	s.connLock.RUnlock()

	// Handle incoming messages (stdin)
	for {
		_, message, readErr := conn.ReadMessage()
		if readErr != nil {
			break
		}

		// Check if this is the authentication message.
		if len(message) > 0 && message[0] == '{' {
			continue // Skip the auth message as it's already handled by the middleware.
		}

		s.runner.WriteInput(string(message))
	}
}

func (s *Server) handleRunnerOutput() {
	for line := range s.runner.GetOutputChan() {
		// Store in buffer
		s.connLock.Lock()
		s.outputBuffer = append(s.outputBuffer, line)
		// Keep buffer size reasonable
		if len(s.outputBuffer) > outputLimit {
			s.outputBuffer = s.outputBuffer[len(s.outputBuffer)-outputLimit:]
		}
		s.connLock.Unlock()

		// Broadcast to all connections.
		s.connLock.RLock()
		deadConns := make([]*websocket.Conn, 0)
		for conn := range s.connections {
			if writeErr := conn.WriteMessage(websocket.TextMessage, []byte(line)); writeErr != nil {
				deadConns = append(deadConns, conn)
			}
		}
		s.connLock.RUnlock()

		if len(deadConns) == 0 {
			continue
		}

		s.connLock.Lock()
		for _, conn := range deadConns {
			if closeErr := conn.Close(); closeErr != nil {
				slog.Default().Warn("Failed to close dead websocket", "err", closeErr)
			}
			delete(s.connections, conn)
		}
		s.connLock.Unlock()
	}
}

func (s *Server) allowedOriginsList() []string {
	if len(s.allowedOrigins) == 0 {
		return []string{"localhost", "127.0.0.1", "::1"}
	}
	return s.allowedOrigins
}

func isAllowedOrigin(r *http.Request, allowedHosts []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := originURL.Hostname()
	if host == "" {
		return false
	}

	for _, allowedHost := range allowedHosts {
		if strings.EqualFold(host, allowedHost) {
			return true
		}
	}

	return false
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	tmpl := template.Must(template.New("index").Parse(htmlTemplate))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Minecraft Server Output</title>
    <style>
        body {
            font-family: monospace;
            background: #1e1e1e;
            color: #d4d4d4;
            padding: 20px;
        }
        #output {
            white-space: pre-wrap;
            padding: 10px;
            background: #2d2d2d;
            border-radius: 5px;
            margin-bottom: 20px;
            height: 400px;
            overflow-y: auto;
        }
        .stdout { color: #6A9955; }
        .stderr { color: #F44747; }
        .disconnected { color: #F44747; font-style: italic; }
        #input-container {
            display: flex;
            gap: 10px;
        }
        #command-input {
            flex-grow: 1;
            padding: 8px;
            background: #2d2d2d;
            border: 1px solid #3d3d3d;
            border-radius: 4px;
            color: #d4d4d4;
            font-family: monospace;
        }
        button {
            padding: 8px 16px;
            background: #0e639c;
            border: none;
            border-radius: 4px;
            color: white;
            cursor: pointer;
        }
        button:hover {
            background: #1177bb;
        }
        .status {
            position: fixed;
            top: 10px;
            right: 10px;
            padding: 5px 10px;
            border-radius: 4px;
            font-size: 12px;
        }
        .status.connected { background: #6A9955; }
        .status.disconnected { background: #F44747; }
    </style>
    <script>
        let ws;
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;

        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            let authKey = localStorage.getItem('authKey');
            if (!authKey) {
                authKey = prompt('Please enter your authentication key:');
                if (authKey) {
                    localStorage.setItem('authKey', authKey);
                } else {
                    console.error('Authentication key is required');
                    return;
                }
            }
            
            // Add auth key as a query parameter
            const wsUrl = new URL(protocol + '//' + window.location.host + '/ws');
            wsUrl.searchParams.append('auth', authKey);
            ws = new WebSocket(wsUrl.toString());

            ws.onopen = function() {
                console.log('Connected to server');
                const status = document.getElementById('status');
                status.textContent = 'Connected';
                status.className = 'status connected';
                reconnectAttempts = 0;
            };

            ws.onclose = function(event) {
                console.log('Disconnected from server:', event.code);
                const status = document.getElementById('status');
                status.textContent = 'Disconnected';
                status.className = 'status disconnected';

                // Check if it was an auth error (code 1008 is policy violation)
                if (event.code === 1008) {
                    localStorage.removeItem('authKey'); // Clear invalid key
                    const output = document.getElementById('output');
                    output.innerHTML +=
                        '<div class="disconnected">Authentication failed. ' +
                        'Please refresh the page to try again.</div>';
                } else if (reconnectAttempts < maxReconnectAttempts) {
                    reconnectAttempts++;
                    setTimeout(connect, 1000 * reconnectAttempts);
                } else {
                    const output = document.getElementById('output');
                    output.innerHTML +=
                        '<div class="disconnected">Connection lost. ' +
                        'Please refresh the page to reconnect.</div>';
                }
            };

            ws.onmessage = function(event) {
                const line = event.data;
                const output = document.getElementById('output');
                const div = document.createElement('div');
                div.className = line.startsWith('[ERR]') ? 'stderr' : 'stdout';
                div.textContent = line;
                output.appendChild(div);
                output.scrollTop = output.scrollHeight;
            };

            ws.onerror = function(error) {
                console.error('WebSocket error:', error);
            };
        }

        function sendCommand() {
            const input = document.getElementById('command-input');
            const command = input.value;
            if (command.trim() === '' || !ws || ws.readyState !== WebSocket.OPEN) return;

            ws.send(command);
            input.value = '';
        }

        document.addEventListener('DOMContentLoaded', function() {
            const input = document.getElementById('command-input');
            input.addEventListener('keypress', function(e) {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    sendCommand();
                }
            });
            connect();
        });
    </script>
</head>
<body>
    <div id="status" class="status disconnected">Disconnected</div>
    <h1>Minecraft Server Output</h1>
    <div id="output"></div>
    <div id="input-container">
        <input type="text" id="command-input" placeholder="Type a command and press Enter">
        <button onclick="sendCommand()">Send</button>
    </div>
</body>
</html>
`
