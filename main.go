// WinAirDrop-Go: 自宅LAN専用の小さなWebDAVサーバー
// iOSショートカットから写真を受け取るための最小構成。
// 設定は config.json (username / password / root / port) のみ。
package main

import (
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/webdav"
)

type Config struct {
	Host     string `json:"host"`     // 例: "0.0.0.0"
	Port     int    `json:"port"`     // 例: 8080
	Root     string `json:"root"`     // 受信フォルダ 例: "C:/Photos"
	Username string `json:"username"`
	Password string `json:"password"`
}

func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Root == "" {
		return nil, fmt.Errorf("root (受信フォルダ) が未設定です")
	}
	if c.Username == "" || c.Password == "" {
		return nil, fmt.Errorf("username / password は必ず設定してください")
	}
	return &c, nil
}

func basicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(u), []byte(user)) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), []byte(pass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="WinAirDrop"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}


// lanIPs は RFC1918 のプライベート IPv4 アドレスを返す (LAN用)。
// Tailscale 等の 100.64/10 はショートカットの接続先にならないので除外する。
func lanIPs() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		if ip.IsPrivate() {
			out = append(out, ip.String())
		}
	}
	return out
}

func main() {
	cfgPath := flag.String("config", "config.json", "設定ファイルのパス")
	flag.Parse()

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("設定エラー: %v", err)
	}
	abs, err := filepath.Abs(cfg.Root)
	if err != nil {
		log.Fatalf("受信フォルダのパス解決に失敗: %v", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		log.Fatalf("受信フォルダを作成できません: %v", err)
	}

	handler := &webdav.Handler{
		FileSystem: webdav.Dir(abs),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("%s %s -> error: %v", r.Method, r.URL.Path, err)
			} else {
				log.Printf("%s %s", r.Method, r.URL.Path)
			}
		},
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("WinAirDrop-Go を起動しました: http://%s/  受信フォルダ: %s", addr, abs)
	// 起動時に現在の設定をまとめて表示する
	ipShown := cfg.Host
	if cfg.Host == "" || cfg.Host == "0.0.0.0" {
		if ips := lanIPs(); len(ips) > 0 {
			ipShown = strings.Join(ips, ", ")
		} else {
			ipShown = "(ipconfig で確認してください)"
		}
	}
	fmt.Println("######################")
	fmt.Printf("IP: %s\n", ipShown)
	fmt.Printf("指定フォルダ: %s\n", abs)
	fmt.Printf("設定ユーザー名: %s\n", cfg.Username)
	fmt.Printf("設定パスワード: %s\n", cfg.Password)
	fmt.Printf("設定ポート: %d\n", cfg.Port)
	fmt.Println("######################")
	if cfg.Host == "" || cfg.Host == "0.0.0.0" {
		if ips := lanIPs(); len(ips) > 0 {
			for _, ip := range ips {
				log.Printf("→ ショートカットのURLには http://%s:%d を使ってください", ip, cfg.Port)
			}
		} else {
			log.Printf("→ LAN内IPが見つかりませんでした。ipconfig で確認してください")
		}
	}
	log.Printf("※ 自宅LAN内でのみ使ってください。インターネットへ公開しないでください。")
	if err := http.ListenAndServe(addr, basicAuth(cfg.Username, cfg.Password, handler)); err != nil {
		log.Fatalf("サーバーエラー: %v", err)
	}
}
