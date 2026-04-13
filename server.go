package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type BazaarResponse struct {
	Success  bool               `json:"success"`
	Products map[string]Product `json:"products"`
}

type Product struct {
	QuickStats Stats `json:"quick_status"`
}

type Stats struct {
	SellPrice float64 `json:"sellPrice"`
	BuyPrice  float64 `json:"buyPrice"`
}

var (
	cache      BazaarResponse
	cacheMutex sync.RWMutex
)

type AuctionResponse struct {
	Success    bool      `json:"success"`
	TotalPages int       `json:"totalPages"`
	Auctions   []Auction `json:"auctions"`
}

type Auction struct {
	ItemName    string `json:"item_name"`
	Tier        string `json:"tier"`
	StartingBid int64  `json:"starting_bid"`
	Bin         bool   `json:"bin"`
	Category    string `json:"category"`
}

var (
	ahCache      []Auction
	ahCacheMutex sync.RWMutex
)

func fetchBazaarData() {
	client := http.Client{Timeout: time.Second * 10}

	for {
		resp, err := client.Get("https://api.hypixel.net/skyblock/bazaar")
		if err == nil {
			var newData BazaarResponse
			if err := json.NewDecoder(resp.Body).Decode(&newData); err == nil {
				cacheMutex.Lock()
				cache = newData
				cacheMutex.Unlock()
				fmt.Println("Bazaar data updated:", time.Now().Format("15:04:05"))
			}
			resp.Body.Close()
		} else {
			fmt.Printf("Update error: %v\n", err)
		}

		time.Sleep(60 * time.Second)
	}
}
func scanAuctionPages(client *http.Client) {
	fmt.Println("Starting AH Scan...")
	resp, err := client.Get("https://api.hypixel.net/skyblock/auctions?page=0")
	if err != nil {
		fmt.Println("AH Error:", err)
		return
	}

	var firstPage AuctionResponse
	json.NewDecoder(resp.Body).Decode(&firstPage)
	resp.Body.Close()

	tempAuctions := []Auction{}
	for i := 0; i < firstPage.TotalPages; i++ {
		pageUrl := fmt.Sprintf("https://api.hypixel.net/skyblock/auctions?page=%d", i)
		pResp, err := client.Get(pageUrl)
		if err != nil {
			continue
		}

		var pageData AuctionResponse
		if err := json.NewDecoder(pResp.Body).Decode(&pageData); err == nil {
			for _, auc := range pageData.Auctions {
				if (auc.Category == "pet" || strings.Contains(auc.ItemName, "[Lvl")) && auc.Bin {
					tempAuctions = append(tempAuctions, auc)
				}
			}
		}
		pResp.Body.Close()
		time.Sleep(10 * time.Millisecond)
	}

	ahCacheMutex.Lock()
	ahCache = tempAuctions
	ahCacheMutex.Unlock()
	fmt.Printf("AH Scan Complete: %d pets cached.\n", len(tempAuctions))
}

func fetchAHData() {
	client := http.Client{Timeout: time.Second * 15}
	for {
		time.Sleep(60 * time.Second)
		scanAuctionPages(&client)
	}
}

func main() {
	fmt.Println("Performing initial data fetch...")
	client := http.Client{Timeout: time.Second * 10}
	resp, err := client.Get("https://api.hypixel.net/skyblock/bazaar")
	if err == nil {
		var newData BazaarResponse
		if err := json.NewDecoder(resp.Body).Decode(&newData); err == nil {
			cache = newData
			fmt.Println("Initial fetch successful.")
		}
		resp.Body.Close()
	}

	fmt.Println("Initializing AH (Synchronous)...")
	scanAuctionPages(&client)

	go fetchBazaarData()
	go fetchAHData()

	http.HandleFunc("/navbar.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/navbar.css")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/main.html")
	})

	http.HandleFunc("/bazaar", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/bazaar.html")
	})

	http.HandleFunc("/ah", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/auction.html")
	})

	http.HandleFunc("/auction", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/auction.html")
	})

	http.HandleFunc("/api/bazaar", func(w http.ResponseWriter, r *http.Request) {
		cacheMutex.RLock()
		defer cacheMutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cache)
	})

	http.HandleFunc("/api/auctions", func(w http.ResponseWriter, r *http.Request) {
		ahCacheMutex.RLock()
		defer ahCacheMutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ahCache)
	})

	fmt.Println("Server starting at http://localhost:8090")
	http.ListenAndServe(":8090", nil)
}
