// file: report_generator.go
package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
	"net/smtp"
	"crypto/tls"

	"github.com/gocql/gocql"
	"github.com/jung-kurt/gofpdf"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"encoding/base64"
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
	PercentChange float64 // (last-first)/first * 100
	AvgPrice      float64
	StdDev        float64 // standard deviation of prices
	Volatility    float64 // std dev of log returns (proxy)
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
func fetchYesterdayData() (MarketData, error) {
	data := make(MarketData)

	iter := session.Query(`
        SELECT coin_id, timestamp, price_usd
        FROM crypto_price_by_coin
        Limit 1000 ALLOW FILTERING`).Consistency(gocql.One).Iter()

	var coinID string
	var ts time.Time
	var price float64

	for iter.Scan(&coinID, &ts, &price) {
		data[coinID] = append(data[coinID], PricePoint{
			Timestamp: ts,
			Price:     price,
		})
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	for k := range data {
		points := data[k]
		sort.Slice(points, func(i, j int) bool {
			return points[i].Timestamp.Before(points[j].Timestamp)
		})
		data[k] = points
	}

	return data, nil
}

// analyzeMarket computes insights (percent change, average, stddev, volatility) per coin.
func analyzeMarket(data MarketData) (ReportInsights, error) {
	insights := make([]Insight, 0, len(data))
	reportDate := time.Now().UTC().Add(-24 * time.Hour)

	for coin, series := range data {
		if len(series) == 0 {
			continue
		}
		first := series[0].Price
		last := series[len(series)-1].Price

		var sum float64
		for _, p := range series {
			sum += p.Price
		}
		avg := sum / float64(len(series))

		var variance float64
		for _, p := range series {
			diff := p.Price - avg
			variance += diff * diff
		}
		stddev := 0.0
		if len(series) > 1 {
			stddev = math.Sqrt(variance / float64(len(series)-1))
		}

		var logRets []float64
		for i := 1; i < len(series); i++ {
			if series[i-1].Price <= 0 || series[i].Price <= 0 {
				continue
			}
			r := math.Log(series[i].Price / series[i-1].Price)
			logRets = append(logRets, r)
		}
		var vol float64
		if len(logRets) > 1 {
			var sumr float64
			for _, r := range logRets {
				sumr += r
			}
			meanr := sumr / float64(len(logRets))
			var vr float64
			for _, r := range logRets {
				dr := r - meanr
				vr += dr * dr
			}
			vol = math.Sqrt(vr / float64(len(logRets)-1))
		}

		pctChange := 0.0
		if first != 0 {
			pctChange = (last - first) / first * 100.0
		}

		insights = append(insights, Insight{
			CoinID:        coin,
			FirstPrice:    first,
			LastPrice:     last,
			PercentChange: pctChange,
			AvgPrice:      avg,
			StdDev:        stddev,
			Volatility:    vol,
			DataPoints:    len(series),
		})
	}

	sort.Slice(insights, func(i, j int) bool {
		return insights[i].PercentChange > insights[j].PercentChange
	})

	top := 5
	if len(insights) < top {
		top = len(insights)
	}
	topGainers := make([]Insight, 0, top)
	topLosers := make([]Insight, 0, top)
	for i := 0; i < top; i++ {
		topGainers = append(topGainers, insights[i])
	}
	for i := 0; i < top; i++ {
		idx := len(insights) - 1 - i
		if idx < 0 {
			break
		}
		topLosers = append(topLosers, insights[idx])
	}

	return ReportInsights{
		Date:        reportDate,
		CoinMetrics: insights,
		TopGainers:  topGainers,
		TopLosers:   topLosers,
	}, nil
}

// createCharts generates PNG line charts for selected coins.
func createCharts(data MarketData, maxCharts int, outDir string) ([]string, error) {
	if maxCharts <= 0 {
		maxCharts = 3
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir outdir: %w", err)
	}

	type coinScore struct {
		coin  string
		score int
	}
	var scores []coinScore
	for coin, series := range data {
		scores = append(scores, coinScore{coin: coin, score: len(series)})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

	selected := make([]string, 0, maxCharts)
	for i := 0; i < len(scores) && len(selected) < maxCharts; i++ {
		selected = append(selected, scores[i].coin)
	}

	paths := []string{}
	for _, coin := range selected {
		series, ok := data[coin]
		if !ok || len(series) == 0 {
			continue
		}

		p := plot.New()
		p.Title.Text = fmt.Sprintf("%s Price (UTC)", coin)
		p.X.Label.Text = "Time"
		p.Y.Label.Text = "Price USD"

		pts := make(plotter.XYs, len(series))
		for i, sp := range series {
			pts[i].X = float64(sp.Timestamp.Unix())
			pts[i].Y = sp.Price
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			log.Printf("plot line error %v", err)
			continue
		}
		line.Color = color.RGBA{R: 0, G: 128, B: 255, A: 255}

		scatter, err := plotter.NewScatter(pts)
		if err == nil {
			scatter.Radius = vg.Points(0.5)
			scatter.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255}
			p.Add(scatter)
		}

		p.Add(line)
		p.X.Tick.Marker = plot.TimeTicks{Format: "15:04"}

		fileName := fmt.Sprintf("%s_chart.png", coin)
		fullPath := filepath.Join(outDir, fileName)
		if err := p.Save(14*vg.Centimeter, 8*vg.Centimeter, fullPath); err != nil {
			log.Printf("failed to save plot for %s: %v", coin, err)
			continue
		}
		paths = append(paths, fullPath)
	}

	return paths, nil
}

// buildPDF constructs a PDF report.
func buildPDF(insights ReportInsights, chartPaths []string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Daily Crypto Market Report", false)
	pdf.SetAuthor("Your Platform Name", false)

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(0, 10, "Daily Crypto Market Report", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	dateStr := insights.Date.Format("2006-01-02 (UTC)")
	pdf.CellFormat(0, 8, fmt.Sprintf("Date: %s", dateStr), "", 1, "C", false, 0, "")
	pdf.Ln(6)

	if len(insights.TopGainers) > 0 {
		g := insights.TopGainers[0]
		summary := fmt.Sprintf("Top Gainer: %s (%.2f%%), Price: %.2f -> %.2f", g.CoinID, g.PercentChange, g.FirstPrice, g.LastPrice)
		pdf.MultiCell(0, 6, summary, "", "C", false)
		pdf.Ln(4)
	}

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, "Top Gainers", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(2)

	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(50, 8, "Coin", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Percent Change", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Avg Price", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Volatility", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	for _, g := range insights.TopGainers {
		pdf.CellFormat(50, 7, g.CoinID, "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.2f%%", g.PercentChange), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.4f", g.AvgPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.6f", g.Volatility), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, "Top Losers", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(2)

	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(50, 8, "Coin", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Percent Change", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Avg Price", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Volatility", "1", 1, "C", true, 0, "")

	for _, g := range insights.TopLosers {
		pdf.CellFormat(50, 7, g.CoinID, "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.2f%%", g.PercentChange), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.4f", g.AvgPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%.6f", g.Volatility), "1", 1, "R", false, 0, "")
	}

	for _, cp := range chartPaths {
		pdf.AddPage()
		if _, err := os.Stat(cp); err != nil {
			continue
		}
		imgOpt := gofpdf.ImageOptions{
			ReadDpi:   true,
			ImageType: "PNG",
		}
		pdf.ImageOptions(cp, 15, 30, 180, 0, false, imgOpt, 0, "")
	}

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, "Snapshot Metrics (Top coins)", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.Ln(2)

	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(40, 8, "Coin", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Change %", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Avg", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "StdDev", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Volatility", "1", 1, "C", true, 0, "")

	maxSnapshot := len(insights.CoinMetrics) // Typically 20
	if len(insights.CoinMetrics) < maxSnapshot {
		maxSnapshot = len(insights.CoinMetrics)
	}
	for i := 0; i < maxSnapshot; i++ {
		c := insights.CoinMetrics[i]
		pdf.CellFormat(40, 6, c.CoinID, "1", 0, "L", false, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.2f%%", c.PercentChange), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.4f", c.AvgPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.6f", c.StdDev), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.6f", c.Volatility), "1", 1, "R", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}

	return buf.Bytes(), nil
}

// generateDailyReportPDF is the main entrypoint for report generation.
func generateDailyReportPDF(tmpDir string) ([]byte, error) {
	data, err := fetchYesterdayData()
	if err != nil {
		return nil, fmt.Errorf("fetch data: %w", err)
	}

	insights, err := analyzeMarket(data)
	if err != nil {
		return nil, fmt.Errorf("analyze: %w", err)
	}

	chartPaths, err := createCharts(data, 3, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("create charts: %w", err)
	}

	pdfBytes, err := buildPDF(insights, chartPaths)
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
			"Please find attached your daily market analysis.", pdfData, "report.pdf"); err != nil {
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
