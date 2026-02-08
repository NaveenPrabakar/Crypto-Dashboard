package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

type OpenAIResponse struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"` // OpenAI returns lowercase "message"
    } `json:"choices"`
}

func callOpenAI(prompt string) (string, error) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("OPENAI_API_KEY not set in environment")
    }

    reqBody := fmt.Sprintf(`{
        "model": "gpt-4o-mini",
        "messages": [{"role":"user", "content": %q}]
    }`, prompt)

    req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer([]byte(reqBody)))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    if resp.StatusCode != http.StatusOK {
        var errResp struct {
            Error struct {
                Message string `json:"message"`
            } `json:"error"`
        }
        _ = json.Unmarshal(body, &errResp)
        msg := errResp.Error.Message
        if msg == "" {
            msg = string(body)
        }
        return "", fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, msg)
    }

    var openaiResp OpenAIResponse
    err = json.Unmarshal(body, &openaiResp)
    if err != nil {
        return "", fmt.Errorf("failed to parse OpenAI response: %w", err)
    }

    if len(openaiResp.Choices) == 0 {
        return "", fmt.Errorf("no choices in OpenAI response")
    }

    cqlQuery := openaiResp.Choices[0].Message.Content
    return cqlQuery, nil
}

// stripCQLFromMarkdown removes markdown code fences so we get raw CQL (e.g. ```cql ... ``` or ``` ... ```).
func stripCQLFromMarkdown(s string) string {
    s = strings.TrimSpace(s)
    for _, prefix := range []string{"```cql", "```sql", "```CQL", "```SQL", "```"} {
        if strings.HasPrefix(s, prefix) {
            s = strings.TrimPrefix(s, prefix)
            break
        }
    }
    s = strings.TrimSpace(s)
    if strings.HasSuffix(s, "```") {
        s = strings.TrimSuffix(s, "```")
    }
    return strings.TrimSpace(s)
}


func handleAsk(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    defer r.Body.Close()
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }

    question := strings.TrimSpace(string(body))
    if question == "" {
        http.Error(w, "Empty question", http.StatusBadRequest)
        return
    }

    now := time.Now().UTC()
    since15 := now.Add(-15 * time.Minute)
    timeContext := fmt.Sprintf(
        "Current UTC time for reference: %s. '15 minutes ago' in UTC: %s. Use these exact timestamp formats in CQL (e.g. for 'last 15 minutes' use timestamp >= '%s').",
        now.Format("2006-01-02 15:04:05"),
        since15.Format("2006-01-02 15:04:05"),
        since15.Format("2006-01-02 15:04:05"),
    )

    aiPrompt := `
You are an assistant that converts natural language questions about the table
"iot_data.crypto_price_by_coin" into valid CQL queries.
The table has columns: coin_id (text), timestamp (timestamp), price_usd (double).
The PRIMARY KEY is (coin_id, timestamp) with CLUSTERING ORDER BY (timestamp DESC).

Rules:
- Return ONLY the raw CQL query as plain text. No markdown, no code blocks, no explanations.
- SELECT must use exactly this column order: coin_id, timestamp, price_usd.
- For time ranges (e.g. "last 15 minutes", "last hour") use WHERE coin_id = '...' AND timestamp >= 'YYYY-MM-DD HH:MM:SS' with the timestamp literal.
- Keyspace is iot_data. Table is crypto_price_by_coin. Always use: FROM iot_data.crypto_price_by_coin
- Add ALLOW FILTERING when using timestamp range or LIMIT. Syntax: ... LIMIT N ALLOW FILTERING; or ... ALLOW FILTERING;
- Coin IDs in the table are lowercase: bitcoin, ethereum, cardano, solana, polkadot, chainlink, etc.

` + timeContext + `

User question: ` + question



    cqlQuery, err := callOpenAI(aiPrompt)
    if err != nil {
        log.Printf("AI ask: OpenAI failed: %v", err)
        writeAskError(w, http.StatusInternalServerError, "AI query generation failed: "+err.Error())
        return
    }

    cqlQuery = stripCQLFromMarkdown(cqlQuery)
    cqlQuery = strings.TrimSpace(cqlQuery)
    if cqlQuery == "" {
        writeAskError(w, http.StatusBadRequest, "No query generated")
        return
    }
    if !strings.HasSuffix(cqlQuery, ";") {
        cqlQuery += ";"
    }
    if !strings.Contains(strings.ToUpper(cqlQuery), "ALLOW FILTERING") {
        cqlQuery = strings.TrimSuffix(cqlQuery, ";")
        cqlQuery += " ALLOW FILTERING;"
    }

    iter := session.Query(cqlQuery).Iter()
    if iter == nil {
        writeAskError(w, http.StatusInternalServerError, "Failed to execute query")
        return
    }

    var results []map[string]interface{}

    for {
        var coinID string
        var ts time.Time
        var price float64

        if !iter.Scan(&coinID, &ts, &price) {
            break
        }

        rowData := map[string]interface{}{
            "coin_id":   coinID,
            "timestamp": ts,
            "price_usd": price,
        }

        results = append(results, rowData)
    }

    if err := iter.Close(); err != nil {
        log.Printf("AI ask: Cassandra query failed: %v", err)
        writeAskError(w, http.StatusBadRequest, "Query execution failed: "+err.Error())
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(results); err != nil {
        log.Printf("AI ask: encode error: %v", err)
    }
}

func writeAskError(w http.ResponseWriter, code int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}
