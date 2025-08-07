package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/gocql/gocql"
    "github.com/robfig/cron/v3"
)

var session *gocql.Session

type CoinGeckoResponse map[string]struct {
    USD float64 `json:"usd"`
}

func main() {
    
    cluster := gocql.NewCluster("host.docker.internal")
    cluster.Keyspace = "iot_data"
    cluster.Consistency = gocql.Quorum

    var err error
    session, err = cluster.CreateSession()
    if err != nil {
        log.Fatalf("Failed to connect to Cassandra: %v", err)
    }
    defer session.Close()

    fmt.Println("Connected to Cassandra")

    
    c := cron.New()
    c.AddFunc("@every 1m", fetchAndStoreCryptoPrices)
    c.Start()

    
    select {}
}

func fetchAndStoreCryptoPrices() {
    url := "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum,ripple,litecoin,cardano,dogecoin,polkadot,bitcoin-cash,binancecoin,chainlink,vechain,tron,monero,solana,avalanche,terra,uniswap,shiba-inu,algorand&vs_currencies=usd"

    resp, err := http.Get(url)
    if err != nil {
        log.Printf("Error fetching data: %v", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("Non-OK HTTP status: %s", resp.Status)
        return
    }

    var prices CoinGeckoResponse
    if err := json.NewDecoder(resp.Body).Decode(&prices); err != nil {
        log.Printf("Error decoding response: %v", err)
        return
    }

    timestamp := time.Now()

    for coinID, data := range prices {
        err := session.Query(`
            INSERT INTO crypto_price_by_coin (coin_id, timestamp, price_usd)
            VALUES (?, ?, ?)`,
            coinID, timestamp, data.USD).Exec()

        if err != nil {
            log.Printf("Error inserting %s data: %v", coinID, err)
        } else {
            log.Printf("Inserted %s price: $%.2f", coinID, data.USD)
        }
    }
}
