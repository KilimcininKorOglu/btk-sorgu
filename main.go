// BTK Site Sorgulama Aracı v3.0.0
// ================================
// Türkiye'de engelli siteleri BTK üzerinden sorgular.
// Gemini API ile CAPTCHA otomatik çözümü yapar.
//
// Kullanım:
//   btk-sorgu <domain>                  Tek site sorgula
//   btk-sorgu --liste sites.txt         Liste ile sorgula
//   btk-sorgu --json <domain>           JSON formatında çıktı
//   btk-sorgu --tui                     TUI modunda çalıştır
//
// Ortam Değişkenleri (.env dosyasından veya sistem ortamından):
//   GEMINI_API_KEY    Google Gemini API anahtarı (ZORUNLU)
//   GEMINI_MODEL      Gemini model adı (varsayılan: gemini-2.5-flash)

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Version bilgisi
// Version is set via -ldflags at build time, defaults to "3.0.0"
var Version = "3.0.0"

// Config yapılandırma sabitleri
type Config struct {
	BaseURL        string
	CaptchaPath    string
	GeminiModel    string
	GeminiAPIURL   string
	GeminiPrompt   string
	MaxRetries     int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
	UserAgent      string
}

// Varsayılan yapılandırma
var config = Config{
	BaseURL:        "https://internet.btk.gov.tr/sitesorgu",
	CaptchaPath:    "/secureimage/captcha.php",
	GeminiModel:    "gemini-2.5-flash",
	GeminiPrompt:   "Read the CAPTCHA text. Reply with ONLY the characters, nothing else.",
	MaxRetries:     3,
	RetryDelay:     1 * time.Second,
	RequestTimeout: 30 * time.Second,
	UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

// QueryResult sorgu sonucu
type QueryResult struct {
	Domain                 string `json:"domain"`
	Timestamp              string `json:"timestamp"`
	Status                 bool   `json:"status"`
	QueryDuration          int64  `json:"queryDuration,omitempty"`
	QueryDurationFormatted string `json:"queryDurationFormatted,omitempty"`
	EngelliMi              bool   `json:"engelliMi"`
	KararTarihi            string `json:"kararTarihi,omitempty"`
	KararNumarasi          string `json:"kararNumarasi,omitempty"`
	DosyaNumarasi          string `json:"dosyaNumarasi,omitempty"`
	DosyaTuru              string `json:"dosyaTuru,omitempty"`
	Mahkeme                string `json:"mahkeme,omitempty"`
	TurkceAciklama         string `json:"turkceAciklama,omitempty"`
	IngilizceAciklama      string `json:"ingilizceAciklama,omitempty"`
	Error                  string `json:"error,omitempty"`
}

// GeminiRequest Gemini API istek yapısı
type GeminiRequest struct {
	Contents         []GeminiContent  `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inline_data,omitempty"`
}

type InlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type GenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// GeminiResponse Gemini API yanıt yapısı
type GeminiResponse struct {
	Candidates     []GeminiCandidate `json:"candidates"`
	PromptFeedback *PromptFeedback   `json:"promptFeedback,omitempty"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type PromptFeedback struct {
	BlockReason string `json:"blockReason,omitempty"`
}

// Global değişkenler
var (
	jsonOutput bool
	client     *http.Client
)

// loadEnvFile .env dosyasını yükler
func loadEnvFile() {
	envPath := filepath.Join(".", ".env")
	file, err := os.Open(envPath)
	if err != nil {
		return // .env dosyası yoksa sessizce devam et
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Tırnak işaretlerini kaldır
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		// Sadece tanımlı değilse ayarla
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// log JSON modunda sessiz, normal modda yazdırır
func log(format string, args ...interface{}) {
	if !jsonOutput {
		fmt.Printf(format+"\n", args...)
	}
}

// formatDuration süreyi okunabilir formata çevirir
func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	} else if ms < 60000 {
		return fmt.Sprintf("%.2fs", float64(ms)/1000)
	}
	minutes := ms / 60000
	seconds := float64(ms%60000) / 1000
	return fmt.Sprintf("%dm %.1fs", minutes, seconds)
}

// isValidDomain domain geçerliliğini kontrol eder
func isValidDomain(domain string) bool {
	if domain == "" {
		return false
	}
	domainRegex := regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	return domainRegex.MatchString(domain)
}

// createHTTPClient HTTP client oluşturur
func createHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar:     jar,
		Timeout: config.RequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("maksimum redirect sayısı aşıldı")
			}
			return nil
		},
	}
}

// getSessionCookies session başlatır
func getSessionCookies() error {
	log("🔗 Session başlatılıyor...")

	req, err := http.NewRequest("GET", config.BaseURL+"/", nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("session başlatılamadı: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("session başlatılamadı: HTTP %d", resp.StatusCode)
	}

	log("✅ Session alındı")
	return nil
}

// getCaptcha CAPTCHA resmini indirir
func getCaptcha() ([]byte, error) {
	timestamp := fmt.Sprintf("0.%08d %d", time.Now().UnixNano()%100000000, time.Now().Unix())
	captchaURL := fmt.Sprintf("%s%s?_CAPTCHA=&t=%s", config.BaseURL, config.CaptchaPath, url.QueryEscape(timestamp))

	log("📥 CAPTCHA indiriliyor...")

	req, err := http.NewRequest("GET", captchaURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", config.UserAgent)
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Referer", config.BaseURL+"/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CAPTCHA indirilemedi: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("CAPTCHA indirilemedi: HTTP %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	imageData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if len(imageData) == 0 {
		return nil, fmt.Errorf("CAPTCHA resmi boş döndü")
	}

	log("✅ CAPTCHA indirildi (%d bytes)", len(imageData))
	return imageData, nil
}

// solveCaptchaWithGemini Gemini API ile CAPTCHA çözer
func solveCaptchaWithGemini(imageData []byte, apiKey string) (string, error) {
	log("🤖 Gemini API ile CAPTCHA çözülüyor...")

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	geminiReq := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: config.GeminiPrompt},
					{InlineData: &InlineData{
						MimeType: "image/png",
						Data:     base64Image,
					}},
				},
			},
		},
		GenerationConfig: GenerationConfig{
			Temperature:     0,
			MaxOutputTokens: 256,
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", config.GeminiModel)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini API isteği başarısız: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 429 {
			return "", fmt.Errorf("Gemini API kota aşıldı")
		} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return "", fmt.Errorf("Gemini API yetkilendirme hatası")
		}
		return "", fmt.Errorf("Gemini API hatası: HTTP %d", resp.StatusCode)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if geminiResp.PromptFeedback != nil && geminiResp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("Gemini güvenlik filtresi: %s", geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("Gemini API boş yanıt döndü")
	}

	candidate := geminiResp.Candidates[0]
	if candidate.FinishReason != "" && candidate.FinishReason != "STOP" {
		return "", fmt.Errorf("Gemini yanıt tamamlanamadı: %s", candidate.FinishReason)
	}

	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini API metin yanıtı vermedi")
	}

	text := candidate.Content.Parts[0].Text
	// Sadece alfanumerik karakterleri al
	captchaRegex := regexp.MustCompile(`[^A-Za-z0-9]`)
	captchaCode := captchaRegex.ReplaceAllString(text, "")

	if len(captchaCode) < 5 || len(captchaCode) > 6 {
		return "", fmt.Errorf("geçersiz CAPTCHA çıktısı: \"%s\" -> \"%s\" (%d karakter)", text, captchaCode, len(captchaCode))
	}

	log("✅ CAPTCHA çözüldü: %s", captchaCode)
	return captchaCode, nil
}

// sorgulaSite BTK sorgusu yapar
func sorgulaSite(domain, captchaCode string) (string, error) {
	log("\n🔍 Sorgulanıyor: %s", domain)

	formData := url.Values{
		"deger":         {domain},
		"ipw":           {""},
		"kat":           {""},
		"tr":            {""},
		"eg":            {""},
		"ayrintili":     {"0"},
		"submit":        {"Sorgula"},
		"security_code": {captchaCode},
	}

	req, err := http.NewRequest("POST", config.BaseURL+"/", strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", config.UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://internet.btk.gov.tr")
	req.Header.Set("Referer", config.BaseURL+"/")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sorgu başarısız: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("sorgu başarısız: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// isCaptchaError CAPTCHA hatası kontrol eder
func isCaptchaError(html string) bool {
	return strings.Contains(html, "Güvenlik kodu hatalı") ||
		strings.Contains(html, "security code") ||
		strings.Contains(html, "Doğrulama kodu")
}

// parseHTML HTML yanıtını parse eder
func parseHTML(html string) QueryResult {
	result := QueryResult{
		Status:    true,
		EngelliMi: false,
	}

	// Türkçe açıklama
	turkceRegex := regexp.MustCompile(`(?i)<span class="yazi2_2">([\s\S]*?)</span>`)
	if match := turkceRegex.FindStringSubmatch(html); len(match) > 1 {
		result.TurkceAciklama = cleanHTML(match[1])
	}

	// İngilizce açıklama
	ingilizceRegex := regexp.MustCompile(`(?i)<span class="yazi3_1">([\s\S]*?)</span>`)
	if match := ingilizceRegex.FindStringSubmatch(html); len(match) > 1 {
		result.IngilizceAciklama = cleanHTML(match[1])
	}

	// Beklenen BTK yanıt formatı bulunamadıysa hata döndür
	if result.TurkceAciklama == "" && result.IngilizceAciklama == "" {
		result.Status = false
		result.Error = "BTK yanıtı tanınamadı, beklenmeyen sayfa formatı"
		return result
	}

	// Engel durumu kontrolü
	if result.TurkceAciklama != "" && strings.Contains(result.TurkceAciklama, "engellenmiştir") {
		result.EngelliMi = true

		// Mahkeme kararları için regex (örn: "2024/12907 D. İş sayılı Ankara 1. Sulh Ceza Hakimliği")
		mahkemeRegex := regexp.MustCompile(`(\d{2}/\d{2}/\d{4}) tarihli ve ((\d+/\d+)\s+([A-Za-zİıÜüÖöÇçŞşĞğ.\s]+?)) sayılı (.+?) kararıyla`)
		if match := mahkemeRegex.FindStringSubmatch(result.TurkceAciklama); len(match) > 5 {
			result.KararTarihi = match[1]
			result.KararNumarasi = strings.TrimSpace(match[2])
			result.DosyaNumarasi = match[3]
			result.DosyaTuru = strings.TrimSpace(match[4])
			result.Mahkeme = match[5]
			return result // Engelli - mahkeme kararı bulundu
		}

		// BTK idari kararları için regex (örn: "490.05.01.2025.-772086 sayılı BTK")
		btkRegex := regexp.MustCompile(`(\d{2}/\d{2}/\d{4}) tarihli ve ([\d.\-]+) sayılı (.+?) kararıyla`)
		if match := btkRegex.FindStringSubmatch(result.TurkceAciklama); len(match) > 3 {
			result.KararTarihi = match[1]
			result.KararNumarasi = match[2]
			result.DosyaNumarasi = match[2]
			result.DosyaTuru = "İdari Karar"
			result.Mahkeme = match[3]
			return result // Engelli - BTK idari kararı bulundu
		}

		// Regex eşleşmese bile engellenmiş - detaylar bulunamadı
		return result
	}

	// Engel yok mesajı kontrolü - sadece engelliMi false ise çalışır
	noBlockPatterns := []string{
		"herhangi bir idari karar bulunmamaktadır",
		"herhangi bir yargı kararı bulunmamaktadır",
	}

	for _, pattern := range noBlockPatterns {
		if strings.Contains(strings.ToLower(result.TurkceAciklama), strings.ToLower(pattern)) {
			result.EngelliMi = false
			result.TurkceAciklama = "Bu site hakkında herhangi bir engel kararı bulunmamaktadır."
			break
		}
	}

	return result
}

// cleanHTML HTML taglerini temizler
func cleanHTML(html string) string {
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	result := tagRegex.ReplaceAllString(html, "")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	return strings.TrimSpace(result)
}

// printResult sonucu güzel formatta yazdırır
func printResult(domain string, result QueryResult, durationMs int64) {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("📌 Domain: %s\n", domain)
	if durationMs > 0 {
		fmt.Printf("⏱️  Sorgu Süresi: %s\n", formatDuration(durationMs))
	}
	fmt.Println(strings.Repeat("═", 60))

	if result.EngelliMi {
		fmt.Println("🚫 Durum: ENGELLİ")
		fmt.Println(strings.Repeat("─", 60))

		if result.KararTarihi != "" {
			fmt.Printf("📅 Karar Tarihi: %s\n", result.KararTarihi)
		}
		if result.DosyaNumarasi != "" {
			fmt.Printf("📋 Dosya Numarası: %s\n", result.DosyaNumarasi)
		}
		if result.DosyaTuru != "" {
			fmt.Printf("📂 Dosya Türü: %s\n", result.DosyaTuru)
		}
		if result.Mahkeme != "" {
			fmt.Printf("⚖️ Mahkeme: %s\n", result.Mahkeme)
		}

		fmt.Println(strings.Repeat("─", 60))

		if result.TurkceAciklama != "" {
			fmt.Println("\n📝 Türkçe Açıklama:")
			fmt.Printf("   %s\n", result.TurkceAciklama)
		}

		if result.IngilizceAciklama != "" {
			fmt.Println("\n📝 English Description:")
			fmt.Printf("   %s\n", result.IngilizceAciklama)
		}
	} else {
		fmt.Println("✅ Durum: ERİŞİLEBİLİR")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println("ℹ️ Bu site hakkında herhangi bir engel kararı bulunmamaktadır.")
	}

	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()
}

// outputJSON JSON formatında çıktı verir
func outputJSON(result QueryResult) {
	result.Timestamp = time.Now().UTC().Format(time.RFC3339)
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}

// outputJSONError JSON formatında hata çıktısı verir
func outputJSONError(domain, message string) {
	result := QueryResult{
		Domain:    domain,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    false,
		Error:     message,
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}

// readDomainsFromFile dosyadan domain listesi okur
func readDomainsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			domains = append(domains, line)
		}
	}

	return domains, scanner.Err()
}

// showHelp yardım mesajını gösterir
func showHelp() {
	fmt.Printf(`
╔════════════════════════════════════════════════════════════╗
║           BTK Site Sorgulama Aracı                         ║
╚════════════════════════════════════════════════════════════╝

v%s

Kullanım:
  btk-sorgu [seçenekler] <domain>

Seçenekler:
  --tui               TUI (Terminal UI) modunda çalıştır
  --liste <dosya>     Dosyadan site listesi oku
  --json              JSON formatında çıktı
  --version, -v       Versiyon bilgisini göster
  --help, -h          Bu yardım mesajını göster

Örnekler:
  btk-sorgu --tui                        # TUI modu
  btk-sorgu discord.com
  btk-sorgu discord.com twitter.com google.com
  btk-sorgu --liste sites.txt
  btk-sorgu --json twitter.com

Ortam Değişkenleri (.env dosyası veya sistem ortamı):
  GEMINI_API_KEY      Google Gemini API anahtarı (ZORUNLU)
  GEMINI_MODEL        Gemini model adı (varsayılan: gemini-2.5-flash)

API Anahtarı Alma:
  https://aistudio.google.com/app/apikey
`, Version)
}

// querySingleDomain tek domain sorgular
func querySingleDomain(domain string, apiKey string) QueryResult {
	startTime := time.Now()

	// Session başlat (cookie jar'da saklanır)
	if err := getSessionCookies(); err != nil {
		return QueryResult{Domain: domain, Status: false, Error: err.Error()}
	}

	// CAPTCHA al
	imageData, err := getCaptcha()
	if err != nil {
		return QueryResult{Domain: domain, Status: false, Error: err.Error()}
	}

	// CAPTCHA çöz
	captchaCode, err := solveCaptchaWithGemini(imageData, apiKey)
	if err != nil {
		return QueryResult{Domain: domain, Status: false, Error: err.Error()}
	}

	// Sorgu yap
	html, err := sorgulaSite(domain, captchaCode)
	if err != nil {
		return QueryResult{Domain: domain, Status: false, Error: err.Error()}
	}

	// CAPTCHA hatası kontrolü
	if isCaptchaError(html) {
		return QueryResult{Domain: domain, Status: false, Error: "CAPTCHA kodu hatalı"}
	}

	// Sonucu parse et
	result := parseHTML(html)
	result.Domain = domain
	result.QueryDuration = time.Since(startTime).Milliseconds()
	result.QueryDurationFormatted = formatDuration(result.QueryDuration)

	return result
}

func main() {
	// .env dosyasını yükle
	loadEnvFile()

	// Ortam değişkenlerinden yapılandırmayı güncelle
	if model := os.Getenv("GEMINI_MODEL"); model != "" {
		config.GeminiModel = model
	}
	if userAgent := os.Getenv("USER_AGENT"); userAgent != "" {
		config.UserAgent = userAgent
	}

	// Komut satırı argümanları
	var (
		listFile    string
		showVersion bool
		showHelpArg bool
		tuiMode     bool
	)

	flag.StringVar(&listFile, "liste", "", "Dosyadan site listesi oku")
	flag.BoolVar(&jsonOutput, "json", false, "JSON formatında çıktı")
	flag.BoolVar(&tuiMode, "tui", false, "TUI modunda çalıştır")
	flag.BoolVar(&showVersion, "version", false, "Versiyon bilgisini göster")
	flag.BoolVar(&showVersion, "v", false, "Versiyon bilgisini göster")
	flag.BoolVar(&showHelpArg, "help", false, "Yardım mesajını göster")
	flag.BoolVar(&showHelpArg, "h", false, "Yardım mesajını göster")
	flag.Parse()

	// Versiyon
	if showVersion {
		fmt.Printf("BTK Site Sorgulama Aracı v%s\n", Version)
		os.Exit(0)
	}

	// Yardım
	if showHelpArg {
		showHelp()
		os.Exit(0)
	}

	// Argüman yoksa varsayılan olarak TUI modunda başla
	if len(flag.Args()) == 0 && listFile == "" && !jsonOutput {
		tuiMode = true
	}

	// API key kontrolü (TUI ve CLI için ortak)
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		if jsonOutput {
			outputJSONError("", "GEMINI_API_KEY ayarlanmamış")
		} else {
			fmt.Fprintln(os.Stderr, "❌ GEMINI_API_KEY ayarlanmamış!")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "   .env dosyası oluşturun:")
			fmt.Fprintln(os.Stderr, "   GEMINI_API_KEY=your_api_key")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "   API anahtarı almak için: https://aistudio.google.com/app/apikey")
		}
		os.Exit(1)
	}

	// TUI modu
	if tuiMode {
		if err := runTUI(apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "❌ TUI hatası: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Domain'leri topla
	var domains []string

	if listFile != "" {
		var err error
		domains, err = readDomainsFromFile(listFile)
		if err != nil {
			if jsonOutput {
				outputJSONError("", fmt.Sprintf("Dosya okunamadı: %s", err.Error()))
			} else {
				fmt.Fprintf(os.Stderr, "❌ Dosya okunamadı: %s\n", err.Error())
			}
			os.Exit(1)
		}
	}

	domains = append(domains, flag.Args()...)

	if len(domains) == 0 {
		if jsonOutput {
			outputJSONError("", "Sorgulanacak domain belirtilmedi")
		} else {
			fmt.Fprintln(os.Stderr, "❌ Sorgulanacak domain belirtilmedi!")
		}
		os.Exit(1)
	}

	// Domain validasyonu
	var validDomains []string
	for _, d := range domains {
		if isValidDomain(d) {
			validDomains = append(validDomains, d)
		} else {
			log("⚠️ Geçersiz domain atlandı: %s", d)
		}
	}

	if len(validDomains) == 0 {
		if jsonOutput {
			outputJSONError("", "Geçerli domain bulunamadı")
		} else {
			fmt.Fprintln(os.Stderr, "❌ Geçerli domain bulunamadı!")
		}
		os.Exit(1)
	}

	// HTTP client oluştur
	client = createHTTPClient()

	log(`
╔════════════════════════════════════════════════════════════╗
║           BTK Site Sorgulama Aracı                         ║
╚════════════════════════════════════════════════════════════╝
`)
	log("📋 Sorgulanacak %d site: %s", len(validDomains), strings.Join(validDomains, ", "))
	log("🤖 Model: %s\n", config.GeminiModel)

	// Sorguları yap
	var results []QueryResult
	blocked := 0
	accessible := 0

	for i, domain := range validDomains {
		var result QueryResult
		var lastErr error

		// Retry mekanizması
		for retry := 0; retry < config.MaxRetries; retry++ {
			if retry > 0 {
				log("🔄 Yeniden deneniyor (%d/%d)...", retry, config.MaxRetries)
				time.Sleep(config.RetryDelay)
				// Yeni client oluştur (yeni session için)
				client = createHTTPClient()
			}

			result = querySingleDomain(domain, apiKey)

			if result.Status {
				lastErr = nil
				break
			}

			lastErr = fmt.Errorf(result.Error)

			// CAPTCHA hatası değilse retry yapma
			if !strings.Contains(result.Error, "CAPTCHA") {
				break
			}
		}

		if lastErr != nil {
			result = QueryResult{
				Domain: domain,
				Status: false,
				Error:  lastErr.Error(),
			}
		}

		results = append(results, result)

		if result.Status {
			if result.EngelliMi {
				blocked++
			} else {
				accessible++
			}

			if jsonOutput {
				outputJSON(result)
			} else {
				printResult(domain, result, result.QueryDuration)
			}
		} else {
			if jsonOutput {
				outputJSONError(domain, result.Error)
			} else {
				fmt.Fprintf(os.Stderr, "❌ %s sorgulanırken hata: %s\n", domain, result.Error)
			}
		}

		// Rate limiting
		if i < len(validDomains)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Özet
	if !jsonOutput && len(validDomains) > 1 {
		fmt.Println("\n📊 ÖZET")
		fmt.Println(strings.Repeat("═", 60))
		fmt.Printf("   🚫 Engelli: %d\n", blocked)
		fmt.Printf("   ✅ Erişilebilir: %d\n", accessible)
		failed := len(validDomains) - blocked - accessible
		if failed > 0 {
			fmt.Printf("   ❓ Hatalı: %d\n", failed)
		}
		fmt.Println(strings.Repeat("═", 60))
	}
}
