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

// PredictResponse is the JSON response for the ML prediction endpoint.
type PredictResponse struct {
	CoinID          string    `json:"coin_id"`
	HorizonMinutes  int       `json:"horizon_minutes"`
	PredictedPrice  float64   `json:"predicted_price"`
	PriceLow        float64   `json:"price_low"`
	PriceHigh       float64   `json:"price_high"`
	Trend           string    `json:"trend"`
	Slope           float64   `json:"slope"`
	DataPoints      int       `json:"data_points"`
	PredictedAt     time.Time `json:"predicted_at"`
	HorizonEndTime  time.Time `json:"horizon_end_time"`
}

// fetchHistoryForML returns historical (timestamp, price) for the coin over the last lookbackMinutes, sorted by time.
func fetchHistoryForML(coinID string, lookbackMinutes int) ([]struct{ T time.Time; P float64 }, error) {
	since := time.Now().Add(-time.Duration(lookbackMinutes) * time.Minute)
	iter := session.Query(`
		SELECT timestamp, price_usd
		FROM crypto_price_by_coin
		WHERE coin_id = ? AND timestamp >= ? ALLOW FILTERING`,
		coinID, since).Consistency(gocql.One).Iter()

	var out []struct{ T time.Time; P float64 }
	var t time.Time
	var p float64
	for iter.Scan(&t, &p) {
		out = append(out, struct{ T time.Time; P float64 }{T: t, P: p})
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	// Sort by time ascending for regression
	sort.Slice(out, func(i, j int) bool { return out[i].T.Before(out[j].T) })
	return out, nil
}

// linearRegression returns slope, intercept, and residual standard error (RSE) from y = slope*x + intercept.
// x and y are the same length; x is typically unix time.
func linearRegression(x, y []float64) (slope, intercept, rse float64) {
	n := float64(len(x))
	if n < 3 {
		return 0, 0, 0
	}
	var sumX, sumY, sumXY, sumXX float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumXX += x[i] * x[i]
	}
	denom := n*sumXX - sumX*sumX
	if math.Abs(denom) < 1e-20 {
		return 0, sumY / n, 0
	}
	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n

	var sumSqErr float64
	for i := range x {
		fit := intercept + slope*x[i]
		sumSqErr += (y[i] - fit) * (y[i] - fit)
	}
	df := n - 2
	if df < 1 {
		df = 1
	}
	rse = math.Sqrt(sumSqErr / df)
	return slope, intercept, rse
}

func trendFromSlope(slope float64) string {
	switch {
	case slope > 0.0001:
		return "Uptrend"
	case slope < -0.0001:
		return "Downtrend"
	default:
		return "Sideways"
	}
}

// getPredict handles GET /predict/{coin_id}?horizon_minutes=60&lookback_minutes=1440
func getPredict(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	coinID := vars["coin_id"]

	horizonMinutes := 60
	if v := r.URL.Query().Get("horizon_minutes"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 10080 {
			horizonMinutes = parsed
		}
	}
	lookbackMinutes := 1440 // 24h default
	if v := r.URL.Query().Get("lookback_minutes"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 30 && parsed <= 43200 {
			lookbackMinutes = parsed
		}
	}

	history, err := fetchHistoryForML(coinID, lookbackMinutes)
	if err != nil {
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}
	if len(history) < 10 {
		http.Error(w, "Not enough data points for prediction", http.StatusBadRequest)
		return
	}

	x := make([]float64, len(history))
	y := make([]float64, len(history))
	for i := range history {
		x[i] = float64(history[i].T.Unix())
		y[i] = history[i].P
	}
	slope, intercept, rse := linearRegression(x, y)

	now := time.Now()
	horizonEnd := now.Add(time.Duration(horizonMinutes) * time.Minute)
	futureUnix := float64(horizonEnd.Unix())
	predictedPrice := intercept + slope*futureUnix

	// Prediction interval: scale RSE by sqrt(1 + 1/n + (x_future - x_mean)^2 / S_xx) for simple interval.
	// Simpler: use pred ± z * rse * sqrt(1 + horizon_scale). horizon_scale grows with horizon so uncertainty widens.
	meanX := 0.0
	for _, v := range x {
		meanX += v
	}
	meanX /= float64(len(x))
	var sxx float64
	for _, v := range x {
		sxx += (v - meanX) * (v - meanX)
	}
	if sxx < 1e-20 {
		sxx = 1
	}
	n := float64(len(x))
	sePred := rse * math.Sqrt(1+1/n+((futureUnix-meanX)*(futureUnix-meanX))/sxx)
	z := 1.96 // ~95% interval
	priceLow := predictedPrice - z*sePred
	priceHigh := predictedPrice + z*sePred
	if priceLow < 0 {
		priceLow = 0
	}

	resp := PredictResponse{
		CoinID:         coinID,
		HorizonMinutes: horizonMinutes,
		PredictedPrice: roundPrice(predictedPrice),
		PriceLow:       roundPrice(priceLow),
		PriceHigh:      roundPrice(priceHigh),
		Trend:          trendFromSlope(slope),
		Slope:          slope,
		DataPoints:     len(history),
		PredictedAt:    now,
		HorizonEndTime: horizonEnd,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func roundPrice(p float64) float64 {
	return math.Round(p*100) / 100
}
