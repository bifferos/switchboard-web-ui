/*
   Control file existence via web interface.
*/

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	TemplatesDir      string `json:"templatesDir"`
	StaticDir         string `json:"staticDir"`
	WidgetDir         string `json:"widgetDir"`
	StateDir          string `json:"stateDir"`
	TokenDir          string `json:"tokenDir"`
	ExternalIpAddress string `json:"externalIpAddress"`
	Port              int    `json:"port"`
}

type ExecutionSpec struct {
	TrueCommand  string `json:"true"`
	FalseCommand string `json:"false"`
}

type FileEntry struct {
	Name    string
	Checked bool
}

// some default locations
const appName = "switchboard-web-ui"

var (
	usrShareDir  = filepath.Join("/usr/share", appName)
	templatesDir = filepath.Join(usrShareDir, "templates")
	staticDir    = filepath.Join(usrShareDir, "static")
	varLibDir    = filepath.Join("/var/lib", appName)
	widgetDir    = filepath.Join(varLibDir, "widget")
	tokenDir     = filepath.Join(varLibDir, "token")
	stateDir     = filepath.Join("/var/lib/switchboard")
)

func main() {

	externalIp := getLocalIP()

	// Some defaults, for missing config.
	cfg := Config{
		TemplatesDir:      templatesDir,
		StaticDir:         staticDir,
		WidgetDir:         widgetDir,
		StateDir:          stateDir,
		TokenDir:          tokenDir,
		ExternalIpAddress: externalIp,
		Port:              6060,
	}
	configPath := flag.String("config", filepath.Join("/etc", appName, "config.json"), "Path to config file")
	register := flag.Bool("register", false, "Register a new user")
	flag.Parse()

	f, err := os.Open(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config file not found: %v\n", err)
	} else {
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding config: %v\n", err)
			os.Exit(1)
		}
	}
	defer f.Close()

	fmt.Println("Using configuration:")
	fmt.Printf("  TemplatesDir: %s\n", cfg.TemplatesDir)
	fmt.Printf("  StaticDir: %s\n", cfg.StaticDir)
	fmt.Printf("  WidgetDir: %v\n", cfg.WidgetDir)
	fmt.Printf("  StateDir: %s\n", cfg.StateDir)
	fmt.Printf("  TokenDir: %s\n", cfg.TokenDir)

	if *register {
		fmt.Printf("Registering a new user.\n")
		registerUser(cfg)
		os.Exit(0)
	}

	tmpl := template.Must(template.ParseFiles(filepath.Join(cfg.TemplatesDir, "index.html")))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleIndex(w, r, tmpl, cfg)
	})
	http.HandleFunc("/toggle", func(w http.ResponseWriter, r *http.Request) {
		handleToggle(w, r, cfg)
	})
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		handleQuery(w, r, cfg)
	})
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		handleRegister(w, r, cfg)
	})

	http.Handle("/favicon.ico", http.FileServer(http.Dir(cfg.StaticDir)))

	fmt.Printf("  Port: %d\n", cfg.Port)
	fmt.Printf("UI running on http://localhost:%d/\n", cfg.Port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request, tmpl *template.Template, cfg Config) {
	if !checkAuth(w, r, cfg) {
		return
	}

	widgetList := listFiles(cfg.WidgetDir)

	var entries []FileEntry
	for _, name := range widgetList {
		filePath := filepath.Join(cfg.StateDir, name)
		checked := false
		if _, err := os.Stat(filePath); err == nil {
			checked = true
		}
		entries = append(entries, FileEntry{Name: name, Checked: checked})
	}
	tmpl.Execute(w, entries)
}

func handleToggle(w http.ResponseWriter, r *http.Request, cfg Config) {
	if !checkAuth(w, r, cfg) {
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.ParseForm()
	name := r.FormValue("name")
	checked := r.FormValue("checked") == "true"

	widgetList := listFiles(cfg.WidgetDir)

	if !contains(widgetList, name) {
		http.Error(w, fmt.Sprintf("name parameter '%s' not in widget list", name), http.StatusBadRequest)
		return
	}

	// Load the widget, check what needs to happen.
	widgetPath := filepath.Join(cfg.WidgetDir, name)
	statInfo, err := os.Stat(widgetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("widget for '%s' does not exist", name), http.StatusBadRequest)
		return
	}

	if statInfo.Size() > 0 {

		data, err := os.ReadFile(widgetPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unable to open '%s'", widgetPath), http.StatusBadRequest)
			return
		}

		var spec ExecutionSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			http.Error(w, fmt.Sprintf("Unable to parse '%s'", widgetPath), http.StatusBadRequest)
			return
		}

		var cmdString string
		if checked {
			cmdString = spec.TrueCommand
		} else {
			cmdString = spec.FalseCommand
		}

		args := strings.Fields(cmdString)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to run command '%s': %v", cmdString, err), http.StatusInternalServerError)
			return
		}
	}

	// Update the state.
	filePath := filepath.Join(cfg.StateDir, name)
	if checked {
		os.MkdirAll(cfg.StateDir, 0755)
		os.WriteFile(filePath, []byte("checked"), 0644)
		fmt.Println("File created:", filePath)
	} else {
		os.Remove(filePath)
		fmt.Println("File removed:", filePath)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func handleQuery(w http.ResponseWriter, r *http.Request, cfg Config) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	queryName := r.URL.Query().Get("name")

	widgetList := listFiles(cfg.WidgetDir)

	if !contains(widgetList, queryName) {
		http.Error(w, fmt.Sprintf("name parameter '%s' not in widget list", queryName), http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(cfg.StateDir, queryName)
	w.Header().Set("Content-Type", "text/plain")
	if _, err := os.Stat(filePath); err == nil {
		w.Write([]byte("true"))
	} else {
		w.Write([]byte("false"))
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request, cfg Config) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}
	tokenPath := filepath.Join(cfg.TokenDir, token)
	if _, err := os.Stat(tokenPath); err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Set cookie for site root
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func listFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	var files []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "~") {
			continue // Skip backup files
		}
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files
}

func getLocalIP() string {
	// Connect to a dummy address to determine the preferred outbound IP
	conn, err := net.Dial("udp", "8.8.8.8:80") // This doesn't actually send data
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func checkAuth(w http.ResponseWriter, r *http.Request, cfg Config) bool {
	cookie, err := r.Cookie("auth_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Access denied: missing or invalid auth token", http.StatusUnauthorized)
		return false
	}
	tokenPath := filepath.Join(cfg.TokenDir, cookie.Value)
	if _, err := os.Stat(tokenPath); err != nil {
		http.Error(w, "Access denied: invalid or expired token", http.StatusUnauthorized)
		return false
	}
	return true
}

func registerUser(cfg Config) {

	if _, err := exec.LookPath("qrencode"); err != nil {
		fmt.Println("qrencode is required for registration")
		os.Exit(1)
	}

	// Generate a random 32-byte token
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate token: %v\n", err)
		os.Exit(1)
	}
	token := hex.EncodeToString(tokenBytes)

	// Ensure tokenDir exists
	if err := os.MkdirAll(cfg.TokenDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create token directory: %v\n", err)
		os.Exit(1)
	}

	// Write token to a file named after the token
	tokenPath := filepath.Join(cfg.TokenDir, token)
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write token file: %v\n", err)
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s:6060/register?token=%s", cfg.ExternalIpAddress, token)
	htmlPath := "register.html"

	// Generate PNG QR code to stdout
	var pngBuf bytes.Buffer
	cmd := exec.Command("qrencode", "-o", "-", "-t", "PNG", "-s", "8", url)
	cmd.Stdout = &pngBuf
	if err := cmd.Run(); err == nil {
		// Encode PNG as base64
		b64 := base64.StdEncoding.EncodeToString(pngBuf.Bytes())
		html := fmt.Sprintf(`
<html>
  <body>
    <h2>Registration URL</h2>
    <p><a href="%s">%s</a></p>
    <img src="data:image/png;base64,%s" alt="QR Code"/>
  </body>
</html>`, url, url, b64)
		os.WriteFile(htmlPath, []byte(html), 0644)
		fmt.Printf("Registration HTML with QR code written to: ./%s\n", htmlPath)
		return
	}
}
