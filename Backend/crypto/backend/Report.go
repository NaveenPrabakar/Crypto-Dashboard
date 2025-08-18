// file: report_generator.go
package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"image/color"
	"log"
	"math"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
	"time"

	"context"
	"encoding/base64"
	"github.com/gocql/gocql"
	"github.com/jung-kurt/gofpdf"
	openai "github.com/sashabaranov/go-openai"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"strings"
)

// PricePoint is a single timestamped price.
type PricePoint struct {
	Timestamp time.Time
	Price     float64
}

// MarketData maps coin_id -> timeseries of PricePoint (sorted by timestamp ascending).
type MarketData map[string][]PricePoint

// Insight holds computed statistics for a single coin.
type Insight struct {
	CoinID        string
	FirstPrice    float64
	LastPrice     float64
	PercentChange float64
	AvgPrice      float64
	StdDev        float64
	Volatility    float64
	MinPrice      float64
	MaxPrice      float64
	MedianPrice   float64
	RangePct      float64
	DataPoints    int
}

// ReportInsights holds the whole day's insights.
type ReportInsights struct {
	Date        time.Time
	CoinMetrics []Insight
	TopGainers  []Insight
	TopLosers   []Insight
}

// fetchYesterdayData queries Cassandra for the previous day's prices and
// returns a MarketData map of coin -> []PricePoint (sorted ascending).
func fetchYesterdayData(session *gocql.Session) (MarketData, error) {
	data := make(MarketData)

	// Get all distinct coin_ids first
	var coinIDs []string
	iter := session.Query(`SELECT DISTINCT coin_id FROM crypto_price_by_coin`).Iter()
	var cid string
	for iter.Scan(&cid) {
		coinIDs = append(coinIDs, cid)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	start := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	// Query each coin individually with range filter on timestamp
	for _, coin := range coinIDs {
		var ts time.Time
		var price float64
		it := session.Query(`
			SELECT timestamp, price_usd
			FROM crypto_price_by_coin
			WHERE coin_id = ? AND timestamp >= ? AND timestamp < ?
			ORDER BY timestamp ASC
		`, coin, start, end).Iter()

		for it.Scan(&ts, &price) {
			data[coin] = append(data[coin], PricePoint{Timestamp: ts, Price: price})
		}
		it.Close()
	}

	return data, nil
}

// analyzeMarket computes insights (percent change, average, stddev, volatility) per coin.
func analyzeMarket(data MarketData) ReportInsights {
	insights := []Insight{}
	reportDate := time.Now().UTC().AddDate(0, 0, -1)

	for coin, series := range data {
		if len(series) == 0 {
			continue
		}

		first := series[0].Price
		last := series[len(series)-1].Price

		sum := 0.0
		for _, p := range series {
			sum += p.Price
		}
		avg := sum / float64(len(series))

		variance := 0.0
		for _, p := range series {
			diff := p.Price - avg
			variance += diff * diff
		}
		stddev := 0.0
		if len(series) > 1 {
			stddev = math.Sqrt(variance / float64(len(series)-1))
		}

		// Volatility = stddev of log returns
		logRets := []float64{}
		for i := 1; i < len(series); i++ {
			if series[i-1].Price > 0 && series[i].Price > 0 {
				logRets = append(logRets, math.Log(series[i].Price/series[i-1].Price))
			}
		}
		vol := 0.0
		if len(logRets) > 1 {
			meanr := 0.0
			for _, r := range logRets {
				meanr += r
			}
			meanr /= float64(len(logRets))
			var vr float64
			for _, r := range logRets {
				dr := r - meanr
				vr += dr * dr
			}
			vol = math.Sqrt(vr / float64(len(logRets)-1))
		}

		pctChange := 0.0
		if first != 0 {
			pctChange = (last - first) / first * 100
		}

		// Min/Max
		minP, maxP := series[0].Price, series[0].Price
		for _, p := range series {
			if p.Price < minP {
				minP = p.Price
			}
			if p.Price > maxP {
				maxP = p.Price
			}
		}

		// Median
		sorted := make([]float64, len(series))
		for i, p := range series {
			sorted[i] = p.Price
		}
		sort.Float64s(sorted)
		median := sorted[len(sorted)/2]

		// Range %
		rangePct := 0.0
		if minP > 0 {
			rangePct = (maxP - minP) / minP * 100
		}

		insights = append(insights, Insight{
			CoinID:        coin,
			FirstPrice:    first,
			LastPrice:     last,
			PercentChange: pctChange,
			AvgPrice:      avg,
			StdDev:        stddev,
			Volatility:    vol,
			MinPrice:      minP,
			MaxPrice:      maxP,
			MedianPrice:   median,
			RangePct:      rangePct,
			DataPoints:    len(series),
		})
	}

	// Sort by gainers
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].PercentChange > insights[j].PercentChange
	})

	top := min(5, len(insights))
	topGainers := append([]Insight{}, insights[:top]...)
	topLosers := append([]Insight{}, reverseSlice(insights)[0:top]...)

	return ReportInsights{
		Date:        reportDate,
		CoinMetrics: insights,
		TopGainers:  topGainers,
		TopLosers:   topLosers,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func reverseSlice(s []Insight) []Insight {
	out := make([]Insight, len(s))
	copy(out, s)
	for i := len(out)/2 - 1; i >= 0; i-- {
		opp := len(out) - 1 - i
		out[i], out[opp] = out[opp], out[i]
	}
	return out
}

func createCharts(data MarketData, maxCharts int, outDir string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	type coinScore struct {
		coin  string
		score int
	}
	var scores []coinScore
	for coin, series := range data {
		scores = append(scores, coinScore{coin, len(series)})
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	selected := []string{}
	for i := 0; i < len(scores) && len(selected) < maxCharts; i++ {
		selected = append(selected, scores[i].coin)
	}

	paths := []string{}
	for _, coin := range selected {
		series := data[coin]
		p := plot.New()
		p.Title.Text = coin + " Price (UTC)"
		p.X.Label.Text = "Time"
		p.Y.Label.Text = "Price USD"

		pts := make(plotter.XYs, len(series))
		for i, sp := range series {
			pts[i].X = float64(sp.Timestamp.Unix())
			pts[i].Y = sp.Price
		}

		line, _ := plotter.NewLine(pts)
		line.Color = color.RGBA{R: 0, G: 128, B: 255, A: 255}
		p.Add(line)

		scatter, _ := plotter.NewScatter(pts)
		scatter.Color = color.Black
		scatter.Radius = vg.Points(0.5)
		p.Add(scatter)

		p.X.Tick.Marker = plot.TimeTicks{Format: "15:04"}

		fileName := fmt.Sprintf("%s_chart.png", coin)
		fullPath := filepath.Join(outDir, fileName)
		if err := p.Save(14*vg.Centimeter, 8*vg.Centimeter, fullPath); err == nil {
			paths = append(paths, fullPath)
		}
	}

	return paths, nil
}

func buildPDF(insights ReportInsights, chartPaths []string, data MarketData) ([]byte, error) {

	openaiClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Daily Crypto Market Report", false)
	pdf.SetAuthor("Crypto Analytics Platform", false)

	// === Footer with page numbers ===
	pdf.AliasNbPages("")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
	})

	// === COVER PAGE ===
	pdf.AddPage()
	addBackground(pdf, "Image/background.jpg")
	pdf.SetFont("Helvetica", "B", 24)
	pdf.CellFormat(0, 15, "Daily Crypto Market Report", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Helvetica", "I", 12)
	pdf.CellFormat(0, 10, fmt.Sprintf("Generated on %s (UTC)", insights.Date.Format("2006-01-02")), "", 1, "C", false, 0, "")
	pdf.Ln(15)

	coverImg := "Image/image.png"
	if _, err := os.Stat(coverImg); err == nil {
		pdf.ImageOptions(coverImg, 55, 80, 100, 0, false,
			gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	}

	// === Helper for section headers ===
	sectionHeader := func(title string) {
		pdf.SetTextColor(40, 70, 130)
		pdf.SetFont("Helvetica", "B", 14)
		pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(3)
	}

	// === DAILY RANGE METRICS ===
	pdf.AddPage()
	addBackground(pdf, "Image/background.jpg")
	sectionHeader("Daily Range Metrics")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(30, 8, "Coin", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Min", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Max", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Median", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Range %", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)

	fill := false
	for _, c := range insights.CoinMetrics {
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		fill = !fill

		pdf.CellFormat(30, 6, c.CoinID, "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.4f", c.MinPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.4f", c.MaxPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.4f", c.MedianPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.2f%%", c.RangePct), "1", 1, "R", true, 0, "")
	}

	rows := [][]string{}
	for _, c := range insights.CoinMetrics {
		rows = append(rows, []string{
			c.CoinID,
			fmt.Sprintf("%.4f", c.MinPrice),
			fmt.Sprintf("%.4f", c.MaxPrice),
			fmt.Sprintf("%.4f", c.MedianPrice),
			fmt.Sprintf("%.2f%%", c.RangePct),
		})
	}
	analysis, _ := generateAnalysisFromOpenAI(context.Background(), openaiClient, "Daily Range Metrics", rows)

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(0, 6, analysis, "", "L", false)

	// === TOP GAINERS PAGE ===
	pdf.AddPage()
	addBackground(pdf, "Image/background.jpg")
	sectionHeader("Top Gainers")

	// Table headers
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(50, 8, "Coin", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Percent Change", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Avg Price", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Volatility", "1", 1, "C", true, 0, "")

	// Table rows (alternating background)
	pdf.SetFont("Helvetica", "", 10)
	fill = false
	for _, g := range insights.TopGainers {
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		fill = !fill

		pdf.CellFormat(50, 7, g.CoinID, "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.2f%%", g.PercentChange), "1", 0, "R", true, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.4f", g.AvgPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.6f", g.Volatility), "1", 1, "R", true, 0, "")
	}

	// After drawing Top Gainers table
	rows = [][]string{}
	for _, g := range insights.TopGainers {
		rows = append(rows, []string{
			g.CoinID,
			fmt.Sprintf("%.2f%%", g.PercentChange),
			fmt.Sprintf("%.4f", g.AvgPrice),
			fmt.Sprintf("%.6f", g.Volatility),
		})
	}

	analysis, _ = generateAnalysisFromOpenAI(context.Background(), openaiClient, "Top Gainers", rows)

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(0, 6, analysis, "", "L", false)

	// === TOP LOSERS PAGE ===
	pdf.AddPage()
	addBackground(pdf, "Image/background.jpg")
	sectionHeader("Top Losers")

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(50, 8, "Coin", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Percent Change", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Avg Price", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Volatility", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	fill = false
	for _, g := range insights.TopLosers {
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		fill = !fill

		pdf.CellFormat(50, 7, g.CoinID, "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.2f%%", g.PercentChange), "1", 0, "R", true, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.4f", g.AvgPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.6f", g.Volatility), "1", 1, "R", true, 0, "")
	}

	rows = [][]string{}
	for _, g := range insights.TopLosers {
		rows = append(rows, []string{
			g.CoinID,
			fmt.Sprintf("%.2f%%", g.PercentChange),
			fmt.Sprintf("%.4f", g.AvgPrice),
			fmt.Sprintf("%.6f", g.Volatility),
		})
	}
	analysis, _ = generateAnalysisFromOpenAI(context.Background(), openaiClient, "Top Losers", rows)

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(0, 6, analysis, "", "L", false)

	// === CHART PAGES ===
	for _, cp := range chartPaths {
		if _, err := os.Stat(cp); err != nil {
			continue
		}
		pdf.AddPage()
		addBackground(pdf, "Image/background.jpg")
		imgOpt := gofpdf.ImageOptions{ReadDpi: true, ImageType: "PNG"}
		pdf.ImageOptions(cp, 15, 40, 180, 0, false, imgOpt, 0, "")
		pdf.Ln(100)
		pdf.SetFont("Helvetica", "I", 10)
		pdf.CellFormat(0, 8, fmt.Sprintf("Price chart: %s", filepath.Base(cp)), "", 1, "C", false, 0, "")

		coinName := strings.TrimSuffix(filepath.Base(cp), "_chart.png")
		analysis, _ := generateChartAnalysis(context.Background(), openaiClient, coinName, data[coinName])
		pdf.Ln(64)
		pdf.MultiCell(0, 6, analysis, "", "L", false)
	}

	// === SNAPSHOT METRICS ===
	pdf.AddPage()
	addBackground(pdf, "background.jpg")
	sectionHeader("Snapshot Metrics (Top Coins)")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(40, 8, "Coin", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Change %", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Avg", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "StdDev", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Volatility", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	fill = false
	for _, c := range insights.CoinMetrics {
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		fill = !fill

		pdf.CellFormat(40, 6, c.CoinID, "1", 0, "L", true, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.2f%%", c.PercentChange), "1", 0, "R", true, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.4f", c.AvgPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.6f", c.StdDev), "1", 0, "R", true, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.6f", c.Volatility), "1", 1, "R", true, 0, "")
	}

	rows = [][]string{}
	for _, c := range insights.CoinMetrics {
		rows = append(rows, []string{
			c.CoinID,
			fmt.Sprintf("%.2f%%", c.PercentChange),
			fmt.Sprintf("%.4f", c.AvgPrice),
			fmt.Sprintf("%.6f", c.StdDev),
			fmt.Sprintf("%.6f", c.Volatility),
		})
	}
	analysis, _ = generateAnalysisFromOpenAI(context.Background(), openaiClient, "Snapshot Metrics", rows)

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(0, 6, analysis, "", "L", false)

	// === OUTPUT ===
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}
	return buf.Bytes(), nil
}

// generateDailyReportPDF is the main entrypoint for report generation.
func generateDailyReportPDF(tmpDir string) ([]byte, error) {
	data, err := fetchYesterdayData(session)
	if err != nil {
		return nil, fmt.Errorf("fetch data: %w", err)
	}

	insights := analyzeMarket(data)
	if err != nil {
		return nil, fmt.Errorf("analyze: %w", err)
	}

	chartPaths, err := createCharts(data, 3, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("create charts: %w", err)
	}

	pdfBytes, err := buildPDF(insights, chartPaths, data)
	if err != nil {
		return nil, fmt.Errorf("build pdf: %w", err)
	}

	for _, p := range chartPaths {
		_ = os.Remove(p)
	}

	return pdfBytes, nil
}

// sendEmailWithAttachment is a stub for email sending (implement as needed).
func sendEmailWithAttachment(to, subject, body string, attachment []byte, filename string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASS")
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	auth := smtp.PlainAuth("", from, password, smtpHost)

	var msg bytes.Buffer
	boundary := "boundary123"

	// Headers
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))

	// Body
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	msg.WriteString(body + "\r\n")

	// Attachment
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString(fmt.Sprintf("Content-Type: application/pdf; name=%q\r\n", filename))
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%q\r\n\r\n", filename))

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment)))
	base64.StdEncoding.Encode(encoded, attachment)
	msg.Write(encoded)
	msg.WriteString("\r\n")
	msg.WriteString(fmt.Sprintf("--%s--", boundary))

	addr := smtpHost + ":" + smtpPort
	conn, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	tlsConfig := &tls.Config{ServerName: smtpHost}
	if err = conn.StartTLS(tlsConfig); err != nil {
		return err
	}

	if err = conn.Auth(auth); err != nil {
		return err
	}

	if err = conn.Mail(from); err != nil {
		return err
	}
	if err = conn.Rcpt(to); err != nil {
		return err
	}

	w, err := conn.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(msg.Bytes()); err != nil {
		return err
	}
	return w.Close()
}

// sendDailyReports fetches emails and sends the daily report.
func sendDailyReports() {
	pdfData, err := generateDailyReportPDF(os.TempDir())
	if err != nil {
		log.Println("Error generating report:", err)
		return
	}

	iter := session.Query(`SELECT email FROM email_subscribers`).Iter()
	var email string
	for iter.Scan(&email) {
		if err := sendEmailWithAttachment(email, "Daily Crypto Report",
			"Dear Subscriber,\n\nAttached is your daily market analysis prepared by the Crypto Dashboard team.\n\nBest regards,\nCrypto Dashboard Team", pdfData, "report.pdf"); err != nil {
			log.Printf("Failed to send email to %s: %v", email, err)
		} else {
			log.Printf("Sent report to %s", email)
		}
	}
	iter.Close()
}


// generateReportHandler HTTP handler serves the PDF report.
func generateReportHandler(w http.ResponseWriter, r *http.Request) {
	sendDailyReports()
	tmpDir := os.TempDir()

	pdfBytes, err := generateDailyReportPDF(tmpDir)
	if err != nil {
		http.Error(w, "Failed to generate report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="daily_crypto_report.pdf"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	if _, err := w.Write(pdfBytes); err != nil {
		log.Printf("Error writing PDF response: %v", err)
	}
}

func generateAnalysisFromOpenAI(ctx context.Context, client *openai.Client, tableName string, rows [][]string) (string, error) {
	// Convert rows into a readable text block
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Table: %s\n", tableName))
	for _, row := range rows {
		b.WriteString(strings.Join(row, " | "))
		b.WriteString("\n")
	}

	prompt := fmt.Sprintf(`You are a financial analyst. 
Analyze the following table and provide a concise professional summary (2-3 sentences) suitable for a crypto market PDF report:

%s`, b.String())

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4o-mini", // fast & cheap; could use gpt-4o if you want more detail
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are a financial analyst that writes concise, professional summaries."},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 200,
	})
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func generateChartAnalysis(ctx context.Context, client *openai.Client, coin string, series []PricePoint) (string, error) {
	if len(series) == 0 {
		return fmt.Sprintf("No price data available for %s.", coin), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Coin: %s\n", coin))
	b.WriteString("Timestamps and Prices:\n")
	for i, p := range series {
		if i >= 20 { // limit rows to avoid giant prompt
			b.WriteString("... (truncated)\n")
			break
		}
		b.WriteString(fmt.Sprintf("%s | %.4f\n", p.Timestamp.Format("15:04"), p.Price))
	}

	prompt := fmt.Sprintf(`You are a financial analyst. 
Given the following intraday price series for %s, summarize the overall trend, volatility, and any key patterns. 
Keep it to 2–3 professional sentences for a financial PDF report.

%s`, coin, b.String())

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are a financial analyst that writes concise, professional summaries."},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 200,
	})
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func addBackground(pdf *gofpdf.Fpdf, patternPath string) {
	if _, err := os.Stat(patternPath); err == nil {
		pdf.ImageOptions(patternPath, 0, 0, 210, 297, false,
			gofpdf.ImageOptions{ImageType: "JPG", ReadDpi: true}, 0, "")
	}
}
