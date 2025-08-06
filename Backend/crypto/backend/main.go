package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"

    "github.com/gocql/gocql"
    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

var session *gocql.Session

type PriceData struct {
    CoinID    string    `json:"coin_id"`
    Timestamp time.Time `json:"timestamp"`
    PriceUSD  float64   `json:"price_usd"`
}

func main() {
    // Connect to Cassandra
cluster := gocql.NewCluster("127.0.0.1")
cluster.Keyspace = "iot_data"
cluster.Consistency = gocql.Quorum

var err error
session, err = cluster.CreateSession()
if err != nil {
    log.Fatalf("Failed to connect to Cassandra: %v", err)
}
defer session.Close()

fmt.Println("Connected to Cassandra")

// Set up router
router := mux.NewRouter()
router.HandleFunc("/latest/{coin_id}", getLatestPrice).Methods("GET")
router.HandleFunc("/history/{coin_id}", getHistory).Methods("GET")

// Setup CORS middleware
c := cors.New(cors.Options{
    AllowedOrigins:   []string{"http://localhost:5173"}, 
    AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
    AllowedHeaders:   []string{"*"},
    AllowCredentials: true,
})

handler := c.Handler(router)

fmt.Println("Server running at :8000")
log.Fatal(http.ListenAndServe(":8000", handler))

}

func getLatestPrice(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    coinID := vars["coin_id"]

    var data PriceData

    err := session.Query(`
        SELECT coin_id, timestamp, price_usd
        FROM crypto_price_by_coin
        WHERE coin_id = ? LIMIT 1`, coinID).
        Consistency(gocql.One).
        Scan(&data.CoinID, &data.Timestamp, &data.PriceUSD)

    if err == gocql.ErrNotFound {
        http.Error(w, "Price data not found", http.StatusNotFound)
        return
    } else if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        log.Printf("Query error: %v", err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

func getHistory(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    coinID := vars["coin_id"]
    minutesParam := r.URL.Query().Get("minutes")

    minutes := 60 
    if minutesParam != "" {
        if parsed, err := strconv.Atoi(minutesParam); err == nil {
            minutes = parsed
        }
    }

    since := time.Now().Add(-time.Duration(minutes) * time.Minute)

    var results []PriceData

    iter := session.Query(`
        SELECT coin_id, timestamp, price_usd
        FROM crypto_price_by_coin
        WHERE coin_id = ? AND timestamp >= ? ALLOW FILTERING`, coinID, since).
        Consistency(gocql.One).
        Iter()

    var data PriceData
    for iter.Scan(&data.CoinID, &data.Timestamp, &data.PriceUSD) {
        
        results = append(results, PriceData{
            CoinID:    data.CoinID,
            Timestamp: data.Timestamp,
            PriceUSD:  data.PriceUSD,
        })
    }

    if err := iter.Close(); err != nil {
        http.Error(w, "Query error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}
