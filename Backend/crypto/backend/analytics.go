package main

import (
    "encoding/json"
    "math"
    "net/http"
    "sort"
    "strconv"
    "time"

    "github.com/gocql/gocql"
    "github.com/gorilla/mux"
)

// Volatility Endpoint
func getVolatility(w http.ResponseWriter, r *http.Request) {
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

    var prices []float64
    var price float64
    for iter.Scan(&price) {
        prices = append(prices, price)
    }

    if err := iter.Close(); err != nil {
        http.Error(w, "Query error", http.StatusInternalServerError)
        return
    }

    if len(prices) < 2 {
        http.Error(w, "Not enough data points", http.StatusBadRequest)
        return
    }

    var sum, mean, variance float64
    for _, p := range prices {
        sum += p
    }
    mean = sum / float64(len(prices))
    for _, p := range prices {
        variance += (p - mean) * (p - mean)
    }
    stddev := math.Sqrt(variance / float64(len(prices)))

    response := map[string]interface{}{
        "coin_id":      coinID,
        "start":        start,
        "end":          end,
        "stddev_price": stddev,
        "mean_price":   mean,
        "data_points":  len(prices),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Trend Endpoint
func getTrend(w http.ResponseWriter, r *http.Request) {
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
        SELECT timestamp, price_usd
        FROM crypto_price_by_coin
        WHERE coin_id = ? AND timestamp >= ? AND timestamp <= ? ALLOW FILTERING`,
        coinID, start, end).Consistency(gocql.One).Iter()

    var timestamp time.Time
    var price float64
    var xValues, yValues []float64

    for iter.Scan(&timestamp, &price) {
        x := float64(timestamp.Unix())
        xValues = append(xValues, x)
        yValues = append(yValues, price)
    }

    if err := iter.Close(); err != nil {
        http.Error(w, "Query error", http.StatusInternalServerError)
        return
    }

    n := len(xValues)
    if n < 2 {
        http.Error(w, "Not enough data points", http.StatusBadRequest)
        return
    }

    var sumX, sumY, sumXY, sumXX float64
    for i := 0; i < n; i++ {
        sumX += xValues[i]
        sumY += yValues[i]
        sumXY += xValues[i] * yValues[i]
        sumXX += xValues[i] * xValues[i]
    }

    slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumXX - sumX*sumX)

    response := map[string]interface{}{
        "coin_id":       coinID,
        "slope":         slope,
        "trend":         trendDescription(slope),
        "data_points":   n,
        "start":         start,
        "end":           end,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func trendDescription(slope float64) string {
    switch {
    case slope > 0.01:
        return "Uptrend"
    case slope < -0.01:
        return "Downtrend"
    default:
        return "Sideways"
    }
}

func getTopMovers(w http.ResponseWriter, r *http.Request) {
    minutes := 1440
    if v := r.URL.Query().Get("minutes"); v != "" {
        if m, err := strconv.Atoi(v); err == nil && m > 0 {
            minutes = m
        }
    }

    since := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)

    iter := session.Query(`SELECT DISTINCT coin_id FROM crypto_price_by_coin`).Iter()
    var coins []string
    var coinID string
    for iter.Scan(&coinID) {
        coins = append(coins, coinID)
    }
    if err := iter.Close(); err != nil {
        http.Error(w, "failed to fetch coin list", http.StatusInternalServerError)
        return
    }

    type CoinChange struct {
        CoinID string  `json:"coin_id"`
        Start  float64 `json:"start_price"`
        End    float64 `json:"end_price"`
        Change float64 `json:"percent_change"`
    }

    var movers []CoinChange

    for _, coin := range coins {
        var startPrice, endPrice float64
        var foundStart, foundEnd bool

        // Price at or before boundary
        err := session.Query(`
            SELECT price_usd
            FROM crypto_price_by_coin
            WHERE coin_id = ? AND timestamp <= ?
            ORDER BY timestamp DESC
            LIMIT 1
        `, coin, since).
            Consistency(gocql.One).
            Scan(&startPrice)
        if err == nil {
            foundStart = true
        }

        // Latest price
        err = session.Query(`
            SELECT price_usd
            FROM crypto_price_by_coin
            WHERE coin_id = ?
            ORDER BY timestamp DESC
            LIMIT 1
        `, coin).
            Consistency(gocql.One).
            Scan(&endPrice)
        if err == nil {
            foundEnd = true
        }

        if foundStart && foundEnd && startPrice > 0 {
            percentChange := ((endPrice - startPrice) / startPrice) * 100
            movers = append(movers, CoinChange{
                CoinID: coin,
                Start:  startPrice,
                End:    endPrice,
                Change: percentChange,
            })
        }
    }

    // Sort by largest absolute change
    sort.Slice(movers, func(i, j int) bool {
        return math.Abs(movers[i].Change) > math.Abs(movers[j].Change)
    })

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(movers)
}
