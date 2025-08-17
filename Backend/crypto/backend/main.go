package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"
    "os"
    "net/smtp"
    "github.com/google/uuid"


    
    "github.com/gorilla/mux"
    "github.com/rs/cors"
    gocqlastra "github.com/datastax/gocql-astra"
    "github.com/gocql/gocql"
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
var err error

    var cluster *gocql.ClusterConfig

    cluster, err = gocqlastra.NewClusterFromURL("https://api.astra.datastax.com", os.Getenv("ASTRA_DB_ID") , os.Getenv("ASTRA_DB_APPLICATION_TOKEN"), 10*time.Second)
    cluster.Keyspace = "iot_data"

    if err != nil {
        log.Fatalf("unable to load cluster %s from astra: %v", os.Getenv("ASTRA_DB_APPLICATION_TOKEN"), err)
    }

    cluster.Timeout = 30 * time.Second
    start := time.Now()
    session, err = gocql.NewSession(*cluster)
    elapsed := time.Now().Sub(start)

    if err != nil {
        log.Fatalf("unable to connect session: %v", err)
    }

    fmt.Println("Making the query now")

    iter := session.Query("SELECT release_version FROM system.local").Iter()

    var version string

    for iter.Scan(&version) {
        fmt.Println(version)
    }

    if err = iter.Close(); err != nil {
        log.Printf("error running query: %v", err)
    }

    fmt.Printf("Connection process took %s", elapsed)

    if err != nil {
        log.Fatalf("unable to connect session: %v", err)
    }


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
router.HandleFunc("/generate-report", generateReportHandler).Methods("GET")
router.HandleFunc("/unsubscribe", removeSubscriber).Methods("POST")
router.HandleFunc("/verify", verifyEmail).Methods("GET")
router.HandleFunc("/ping", pingHandler).Methods("GET", "HEAD")
router.HandleFunc("/verifyDel", verifyEmailDel).Methods("GET")











c := cors.New(cors.Options{
    AllowedOrigins:   []string{"*"}, 
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

    var data PriceData
    err = session.Query(`
        SELECT coin_id, timestamp, price_usd
        FROM crypto_price_by_coin
        WHERE coin_id = ? AND timestamp <= ?
        ORDER BY timestamp DESC
        LIMIT 1 ALLOW FILTERING`,
        coinID, ts).
        Consistency(gocql.One).
        Scan(&data.CoinID, &data.Timestamp, &data.PriceUSD)

    if err == gocql.ErrNotFound {
        http.Error(w, "Price data not found", http.StatusNotFound)
        return
    } else if err != nil {
        http.Error(w, "Query error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
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

    
    token := uuid.New().String()
    createdAt := time.Now()

    
    err := session.Query(`
        INSERT INTO iot_data.staging_subscribers ("token", email, created_at)
        VALUES (?, ?, ?)`,
        token, sub.Email, createdAt,
    ).Exec()
    if err != nil {
        log.Printf("Error inserting subscriber into staging: %v", err)
        http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
        return
    }

    
    verificationLink := fmt.Sprintf("https://crypto-dashboard-dkzi.onrender.com/verify?token=%s", token)
    if err := sendVerificationEmail(sub.Email, verificationLink); err != nil {
        log.Printf("Error sending verification email: %v", err)
        http.Error(w, "Failed to send verification email", http.StatusInternalServerError)
        return
    }

    // Respond success
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message": "Verification email sent! Please check your inbox.",
        "email":   sub.Email,
    })
}

func sendVerificationEmail(to, link string) error {
	from := os.Getenv("SMTP_EMAIL")
	pass := os.Getenv("SMTP_PASS")

	msg := []byte(fmt.Sprintf(`Subject: Confirm Your Subscription - Crypto Dashboard

Hello,

You (or someone using your email) requested to subscribe to daily crypto market reports
from our Crypto Dashboard.

Please confirm your subscription by clicking the link below:
%s

If you did not request this, you can safely ignore this email and no action will be taken.

Thank you,  
Crypto Dashboard Team
`, link))

	auth := smtp.PlainAuth("", from, pass, "smtp.gmail.com")
	return smtp.SendMail("smtp.gmail.com:587", auth, from, []string{to}, msg)
}

func sendVerificationEmailDel(to, link string) error {
	from := os.Getenv("SMTP_EMAIL")
	pass := os.Getenv("SMTP_PASS")

	msg := []byte(fmt.Sprintf(`Subject: Confirm Unsubscribe - Crypto Dashboard

Hello,

We received a request to unsubscribe you from daily crypto market reports
from our Crypto Dashboard.

If you would like to stop receiving these reports, please confirm by clicking the link below:
%s

If you did not request this change, you can safely ignore this email and your subscription will remain active.

Thank you,  
Crypto Dashboard Team
`, link))

	auth := smtp.PlainAuth("", from, pass, "smtp.gmail.com")
	return smtp.SendMail("smtp.gmail.com:587", auth, from, []string{to}, msg)
}

func verifyEmail(w http.ResponseWriter, r *http.Request) {
    
    token := r.URL.Query().Get("token")
    if token == "" {
        http.Error(w, "Token is required", http.StatusBadRequest)
        return
    }

   
    var email string
    var createdAt time.Time
    err := session.Query(`
        SELECT email, created_at
        FROM iot_data.staging_subscribers
        WHERE "token" = ?`,
        token,
    ).Scan(&email, &createdAt)

    if err != nil {
        log.Printf("Invalid or expired token: %v", err)
        http.Error(w, "Invalid or expired token", http.StatusBadRequest)
        return
    }

    
    if time.Since(createdAt) > 30*time.Minute {
        http.Error(w, "Token expired", http.StatusBadRequest)
        return
    }

    
    err = session.Query(`
        INSERT INTO iot_data.email_subscribers (email, subscribed_at)
        VALUES (?, ?)`,
        email, time.Now(),
    ).Exec()
    if err != nil {
        log.Printf("Error adding verified email: %v", err)
        http.Error(w, "Failed to verify email", http.StatusInternalServerError)
        return
    }

    
    err = session.Query(`
        DELETE FROM iot_data.staging_subscribers
        WHERE "token" = ?`,
        token,
    ).Exec()
    if err != nil {
        log.Printf("Error deleting token from staging: %v", err)
    }

    
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte("Email verified!"))
}


func removeSubscriber(w http.ResponseWriter, r *http.Request) {
    var sub Subscriber

    if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    if sub.Email == "" {
        http.Error(w, "Email is required", http.StatusBadRequest)
        return
    }

    
    token := uuid.New().String()
    createdAt := time.Now()

    
    err := session.Query(`
        INSERT INTO iot_data.staging_subscribers ("token", email, created_at)
        VALUES (?, ?, ?)`,
        token, sub.Email, createdAt,
    ).Exec()
    if err != nil {
        log.Printf("Error inserting subscriber into staging: %v", err)
        http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
        return
    }

    verificationLink := fmt.Sprintf("https://crypto-dashboard-dkzi.onrender.com/verifyDel?token=%s", token)
    if err := sendVerificationEmailDel(sub.Email, verificationLink); err != nil {
        log.Printf("Error sending verification email: %v", err)
        http.Error(w, "Failed to send verification email", http.StatusInternalServerError)
        return
    }

    // Respond success
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message": "Verification email sent! Please check your inbox.",
        "email":   sub.Email,
    })
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        w.Write([]byte("pong"))
    case http.MethodHead:
        w.WriteHeader(http.StatusOK)
    default:
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
    }
}




func verifyEmailDel(w http.ResponseWriter, r *http.Request) {
    
    token := r.URL.Query().Get("token")
    if token == "" {
        http.Error(w, "Token is required", http.StatusBadRequest)
        return
    }

   
    var email string
    var createdAt time.Time
    err := session.Query(`
        SELECT email, created_at
        FROM iot_data.staging_subscribers
        WHERE "token" = ?`,
        token,
    ).Scan(&email, &createdAt)

    if err != nil {
        log.Printf("Invalid or expired token: %v", err)
        http.Error(w, "Invalid or expired token", http.StatusBadRequest)
        return
    }

    
    if time.Since(createdAt) > 30*time.Minute {
        http.Error(w, "Token expired", http.StatusBadRequest)
        return
    }

    
    err = session.Query(`
        DELETE FROM iot_data.email_subscribers Where email = ?`,
        email,
    ).Exec()
    if err != nil {
        log.Printf("Error adding verified email: %v", err)
        http.Error(w, "Failed to remove email", http.StatusInternalServerError)
        return
    }
    
    err = session.Query(`
        DELETE FROM iot_data.staging_subscribers
        WHERE "token" = ?`,
        token,
    ).Exec()
    if err != nil {
        log.Printf("Error deleting token from staging: %v", err)
    }

    
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte("Subscription removal verified!"))
}







