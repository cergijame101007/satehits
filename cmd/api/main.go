package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// ルートパスへのハンドラを登録
	http.HandleFunc("/", handleRoot)

	// サーバー起動
	addr := ":8080"
	log.Printf("Server starting on %s", addr)
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// GETメソッド以外は405 Method Not Allowedを返す
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// "Welcome" を返す
	fmt.Fprint(w, "Welcome to the Go API")
}