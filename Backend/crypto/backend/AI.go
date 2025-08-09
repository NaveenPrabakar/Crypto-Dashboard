package main

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
	"encoding/json" 
	"strings"  
    "io"
    "log"
    "time"
)

type OpenAIResponse struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
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

aiPrompt := `
You are an assistant that converts natural language questions about the table
"iot_data.crypto_price_by_coin" into valid CQL queries.
The table has columns: coin_id (text), timestamp (timestamp), price_usd (double).
The PRIMARY KEY is (coin_id, timestamp) with timestamp in DESC order.

Return ONLY the raw CQL query as plain text (no markdown, no explanations).
The query must be safe, include ALLOW FILTERING where needed, and return actual data from the table.
The query must always select all three columns: coin_id, timestamp, price_usd.

IMPORTANT: If the query includes a LIMIT clause, ALLOW FILTERING must come immediately after the LIMIT clause.
The correct syntax is: 
  SELECT ... FROM ... WHERE ... LIMIT ... ALLOW FILTERING;

User question: ` + question



    cqlQuery, err := callOpenAI(aiPrompt)
    if err != nil {
        http.Error(w, "AI query generation failed", http.StatusInternalServerError)
        return
    }

    cqlQuery = strings.TrimSpace(cqlQuery)
    if !strings.Contains(strings.ToUpper(cqlQuery), "ALLOW FILTERING") {
        cqlQuery = strings.TrimSuffix(cqlQuery, ";")
        cqlQuery += " ALLOW FILTERING;"
    }

    

    iter := session.Query(cqlQuery).Iter()
    if iter == nil {
        http.Error(w, "Failed to execute query", http.StatusInternalServerError)
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
        
        http.Error(w, "Cassandra query execution failed", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)

    if err := json.NewEncoder(w).Encode(results); err != nil {
       
        http.Error(w, "Failed to write response", http.StatusInternalServerError)
        return
    }
}
