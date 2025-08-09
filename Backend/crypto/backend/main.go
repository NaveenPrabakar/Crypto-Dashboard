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

type Subscriber struct {
    Email string `json:"email"`
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
router.HandleFunc("/average/{coin_id}", getAveragePrice).Methods("GET")
router.HandleFunc("/at/{coin_id}", getPriceAtTime).Methods("GET")
router.HandleFunc("/range/{coin_id}", getPriceRange).Methods("GET")
router.HandleFunc("/coins", getAvailableCoins).Methods("GET")
router.HandleFunc("/volatility/{coin_id}", getVolatility).Methods("GET")
router.HandleFunc("/trend/{coin_id}", getTrend).Methods("GET")
router.HandleFunc("/top-movers", getTopMovers).Methods("GET")
router.HandleFunc("/ask", handleAsk).Methods("POST")
router.HandleFunc("/subscribe", addSubscriber).Methods("POST")








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

    minutes := 60 // default 1 hour
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


func getAveragePrice(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    coinID := vars["coin_id"]

    startStr := r.URL.Query().Get("start") 
    endStr := r.URL.Query().Get("end")

    start, err := time.Parse(time.RFC3339, startStr)
    if err != nil {
        http.Error(w, "Invalid start time", http.StatusBadRequest)
        return
    }

    end, err := time.Parse(time.RFC3339, endStr)
    if err != nil {
        http.Error(w, "Invalid end time", http.StatusBadRequest)
        return
    }

    iter := session.Query(`
        SELECT price_usd
        FROM crypto_price_by_coin
        WHERE coin_id = ? AND timestamp >= ? AND timestamp <= ? ALLOW FILTERING`,
        coinID, start, end).Consistency(gocql.One).Iter()

    var sum float64
    var count int
    var price float64

    for iter.Scan(&price) {
        sum += price
        count++
    }

    if err := iter.Close(); err != nil {
        http.Error(w, "Query error", http.StatusInternalServerError)
        return
    }

    if count == 0 {
        http.Error(w, "No data found", http.StatusNotFound)
        return
    }

    average := sum / float64(count)
    response := map[string]interface{}{
        "coin_id":      coinID,
        "average":      average,
        "data_points":  count,
        "start":        start,
        "end":          end,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getPriceAtTime(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    coinID := vars["coin_id"]
    tsStr := r.URL.Query().Get("timestamp") 

    ts, err := time.Parse(time.RFC3339, tsStr)
    if err != nil {
        http.Error(w, "Invalid timestamp", http.StatusBadRequest)
        return
    }

    iter := session.Query(`
        SELECT coin_id, timestamp, price_usd
        FROM crypto_price_by_coin
        WHERE coin_id = ? AND timestamp <= ? ALLOW FILTERING
        LIMIT 1`, coinID, ts).
        Consistency(gocql.One).
        Iter()

    var data PriceData
    if iter.Scan(&data.CoinID, &data.Timestamp, &data.PriceUSD) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(data)
    } else {
        http.Error(w, "Price data not found", http.StatusNotFound)
    }

    iter.Close()
}

func getPriceRange(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    coinID := vars["coin_id"]

    startStr := r.URL.Query().Get("start")
    endStr := r.URL.Query().Get("end")

    start, err := time.Parse(time.RFC3339, startStr)
    if err != nil {
        http.Error(w, "Invalid start time", http.StatusBadRequest)
        return
    }

    end, err := time.Parse(time.RFC3339, endStr)
    if err != nil {
        http.Error(w, "Invalid end time", http.StatusBadRequest)
        return
    }

    iter := session.Query(`
        SELECT price_usd
        FROM crypto_price_by_coin
        WHERE coin_id = ? AND timestamp >= ? AND timestamp <= ? ALLOW FILTERING`,
        coinID, start, end).
        Consistency(gocql.One).
        Iter()

    var price float64
    var min, max float64
    first := true

    for iter.Scan(&price) {
        if first {
            min, max = price, price
            first = false
        } else {
            if price < min {
                min = price
            }
            if price > max {
                max = price
            }
        }
    }

    if err := iter.Close(); err != nil {
        http.Error(w, "Query error", http.StatusInternalServerError)
        return
    }

    if first {
        http.Error(w, "No data found", http.StatusNotFound)
        return
    }

    response := map[string]interface{}{
        "coin_id":  coinID,
        "min":      min,
        "max":      max,
        "start":    start,
        "end":      end,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}


func getAvailableCoins(w http.ResponseWriter, r *http.Request) {
    iter := session.Query(`SELECT DISTINCT coin_id FROM crypto_price_by_coin`).Iter()

    var coinID string
    var coins []string

    for iter.Scan(&coinID) {
        coins = append(coins, coinID)
    }

    if err := iter.Close(); err != nil {
        http.Error(w, "Query error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(coins)
}



func addSubscriber(w http.ResponseWriter, r *http.Request) {
    var sub Subscriber

    if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    if sub.Email == "" {
        http.Error(w, "Email is required", http.StatusBadRequest)
        return
    }

    err := session.Query(`
        INSERT INTO email_subscribers (email, subscribed_at)
        VALUES (?, ?)`,
        sub.Email, time.Now(),
    ).Exec()

    if err != nil {
        log.Printf("Error inserting subscriber: %v", err)
        http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message": "Subscription successful",
        "email":   sub.Email,
    })
}

