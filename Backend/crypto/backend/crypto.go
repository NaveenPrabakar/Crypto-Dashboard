package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    "os"

    
    "github.com/robfig/cron/v3"
    gocqlastra "github.com/datastax/gocql-astra"
    "github.com/gocql/gocql"
   
)

var session *gocql.Session

type CoinGeckoResponse map[string]struct {
    USD float64 `json:"usd"`
}

func main() {
    
    var err error

    var cluster *gocql.ClusterConfig

    cluster, err = gocqlastra.NewClusterFromURL("https://api.astra.datastax.com", os.Getenv("ASTRA_DB_ID"), os.Getenv("ASTRA_DB_APPLICATION_TOKEN"), 10*time.Second)
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

    
    c := cron.New()
    c.AddFunc("@every 10m", fetchAndStoreCryptoPrices)
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
