# BTK Site Sorgulama Aracı v3.0.2

Türkiye'de erişime engellenen web sitelerini [BTK Site Bilgileri Sorgu Sayfası](https://internet.btk.gov.tr/sitesorgu) üzerinden otomatik olarak sorgulayan Go aracı.

**Özellikler:**

- Google Gemini AI ile otomatik CAPTCHA çözümü
- Varsayılan TUI (Terminal UI) modu - exe çift tıklamayla açılır
- CLI modu ile tek veya çoklu site sorgulama
- Mahkeme ve BTK idari kararlarını destekler
- Dosyadan liste okuma
- JSON formatında temiz çıktı desteği
- Her sorgu için süre ölçümü
- TUI'da sorgu geçmişi ve yenileme
- Otomatik yeniden deneme (3x)
- 30 saniye HTTP timeout

---

## Kurulum

### Gereksinimler

- Go 1.24+ (derleme için) veya hazır binary
- Google Gemini API anahtarı (**zorunlu**)

### 1. Dosyaları İndir

```bash
git clone https://github.com/KilimcininKorOglu/btk-sorgu.git
cd btk-sorgu
```

### 2. Derle

```bash
# Linux/macOS (make ile)
make build

# Windows (build.bat ile)
build.bat build

# Tum platformlar icin cross-compile
make build-all
```

### 3. Gemini API Anahtarı Al

1. [Google AI Studio](https://aistudio.google.com/app/apikey) adresine gidin
2. Google hesabınızla giriş yapın
3. **"Create API Key"** butonuna tıklayın
4. API anahtarını kopyalayın

### 4. Ortam Değişkenlerini Ayarla

**Seçenek 1: `.env` Dosyası (Önerilen)**

Proje klasöründe `.env` dosyası oluşturun:

```env
GEMINI_API_KEY=AIzaSy...your_api_key_here
GEMINI_MODEL=gemini-2.5-flash
```

> `.env.example` dosyasını `.env` olarak kopyalayabilirsiniz.

**Seçenek 2: Sistem Ortam Değişkenleri**

Windows (CMD):

```cmd
set GEMINI_API_KEY=AIzaSy...your_api_key_here
```

Windows (PowerShell):

```powershell
$env:GEMINI_API_KEY="AIzaSy...your_api_key_here"
```

Linux/macOS:

```bash
export GEMINI_API_KEY=AIzaSy...your_api_key_here
```

### Ortam Değişkenleri

| Değişken         | Zorunlu | Varsayılan         | Açıklama                   |
|------------------|---------|--------------------|----------------------------|
| `GEMINI_API_KEY` | Evet    | -                  | Google Gemini API anahtarı |
| `GEMINI_MODEL`   | Hayır   | `gemini-2.5-flash` | Kullanılacak Gemini modeli |
| `USER_AGENT`     | Hayır   | Chrome 120         | Özel User-Agent string     |

---

## Kullanım

### TUI Modu (Varsayılan)

Argüman olmadan çalıştırıldığında veya exe'ye çift tıklandığında TUI modu açılır:

```bash
btk-sorgu            # TUI modu (varsayılan)
btk-sorgu --tui      # TUI modu (açık)
```

**TUI Klavye Kısayolları:**

- `Enter` - Sorgula / Geçmişten seçili öğeyi yenile
- `Tab` - Input ve geçmiş tablosu arasında geçiş
- `↑↓` - Geçmiş tablosunda gezinme
- `Ctrl+D` - Geçmişi temizle
- `Esc` - Giriş ekranına dön
- `Q` - Çıkış (geçmiş ve sonuç ekranında)
- `Ctrl+C` - Çıkış (her ekranda)

### CLI Modu

```bash
# Tek site sorgula
btk-sorgu discord.com

# Birden fazla site
btk-sorgu discord.com twitter.com google.com

# Dosyadan liste okuma
btk-sorgu --liste sites.txt

# JSON formatında çıktı
btk-sorgu --json discord.com

# Dosyaya kaydet
btk-sorgu --json discord.com > sonuc.json
```

### Versiyon ve Yardım

```bash
btk-sorgu --version
btk-sorgu --help
```

---

## Komut Satırı Seçenekleri

| Seçenek           | Açıklama                                    |
|-------------------|---------------------------------------------|
| *(argüman yok)*   | TUI modunda çalıştır (varsayılan)           |
| `--tui`           | TUI modunda çalıştır (açık)                 |
| `--liste <dosya>` | Dosyadan site listesi oku (CLI modu)        |
| `--json`          | JSON formatında çıktı (temiz, progress yok) |
| `--version`, `-v` | Versiyon bilgisini göster                   |
| `--help`, `-h`    | Yardım mesajını göster                      |

---

## Liste Dosyası Formatı

`sites.txt` örneği:

```text
# Yorum satırları # ile başlar
discord.com
twitter.com
google.com
```

---

## Örnek Çıktılar

### Engellenmiş Site

```
════════════════════════════════════════════════════════════
📌 Domain: discord.com
⏱️  Sorgu Süresi: 2.04s
════════════════════════════════════════════════════════════
🚫 Durum: ENGELLİ
────────────────────────────────────────────────────────────
📅 Karar Tarihi: 09/10/2024
📋 Dosya Numarası: 2024/12907
📂 Dosya Türü: D. İş
⚖️  Mahkeme: Ankara 1. Sulh Ceza Hakimliği
────────────────────────────────────────────────────────────

📝 Türkçe Açıklama:
   discord.com, 09/10/2024 tarihli ve 2024/12907 D. İş sayılı
   Ankara 1. Sulh Ceza Hakimliği kararıyla erişime engellenmiştir.

📝 English Description:
   discord.com has been blocked by the decision dated 09/10/2024
   and numbered 2024/12907 D. İş of Ankara 1. Sulh Ceza Hakimliği.
════════════════════════════════════════════════════════════
```

### Erişilebilir Site

```
════════════════════════════════════════════════════════════
📌 Domain: google.com
⏱️  Sorgu Süresi: 1.86s
════════════════════════════════════════════════════════════
✅ Durum: ERİŞİLEBİLİR
────────────────────────────────────────────────────────────
ℹ️  Bu site hakkında herhangi bir engel kararı bulunmamaktadır.
════════════════════════════════════════════════════════════
```

### JSON Çıktı

```json
{
  "domain": "discord.com",
  "timestamp": "2025-12-06T12:47:32Z",
  "status": true,
  "queryDuration": 2040,
  "queryDurationFormatted": "2.04s",
  "engelliMi": true,
  "kararTarihi": "09/10/2024",
  "kararNumarasi": "2024/12907 D. İş",
  "dosyaNumarasi": "2024/12907",
  "dosyaTuru": "D. İş",
  "mahkeme": "Ankara 1. Sulh Ceza Hakimliği",
  "turkceAciklama": "discord.com, 09/10/2024 tarihli ve 2024/12907 D. İş sayılı Ankara 1. Sulh Ceza Hakimliği kararıyla erişime engellenmiştir.",
  "ingilizceAciklama": "discord.com has been blocked by the decision dated 09/10/2024 and numbered 2024/12907 D. İş of Ankara 1. Sulh Ceza Hakimliği."
}
```

---

## Sorun Giderme

### "GEMINI_API_KEY ayarlanmamış"

`.env` dosyası oluşturun veya ortam değişkeni ayarlayın.

### "CAPTCHA çözülemedi"

- Gemini API anahtarınızın geçerli olduğundan emin olun
- Araç otomatik olarak 3 kez yeniden dener

### "Session başlatılamadı" hatası

- İnternet bağlantınızı kontrol edin
- BTK sunucusu geçici olarak erişilemez olabilir

### "İstek zaman aşımı" hatası

- Ağ bağlantınızı kontrol edin
- 30 saniye içinde yanıt alınamadı

---

## Teknik Detaylar

### Nasıl Çalışır?

1. **Session Başlatma:** BTK ana sayfasına GET isteği, session cookie'leri alınır
2. **CAPTCHA İndirme:** Session cookie'leri ile CAPTCHA resmi indirilir
3. **CAPTCHA Çözme:** Resim base64'e çevrilip Gemini API'ye gönderilir
4. **Sorgu Gönderme:** POST isteği ile site sorgulanır
5. **Sonuç Parse:** HTML yanıtından engel bilgileri regex ile çıkarılır

### Dosya Yapısı

```
btk-sorgu/
├── main.go            # Ana CLI mantığı, HTTP client, Gemini API
├── tui.go             # Terminal UI (Bubble Tea)
├── go.mod             # Go modülü
├── go.sum             # Bağımlılıklar
├── Makefile           # Build, test, lint hedefleri
├── build.bat          # Windows build script
├── .goreleaser.yml    # GoReleaser yapilandirmasi
├── CHANGELOG.md       # Sürüm değişiklik günlüğü
├── .env               # API anahtarları (oluşturulmalı)
├── .env.example       # Örnek .env
├── history.json       # TUI sorgu geçmişi (otomatik)
├── .github/workflows  # GitHub Actions CI/CD
└── README.md
```

### Bağımlılıklar

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - TUI styling
- `github.com/charmbracelet/bubbles` - TUI components

---

## Maliyet

- **Gemini API:** Ücretsiz tier günde 60 istek/dakika destekler
- Her CAPTCHA çözümü = 1 API isteği
- Pratikte sınırsız kullanım

---

## Yasal Uyarı

Bu araç yalnızca **eğitim ve bilgilendirme** amaçlıdır. BTK'nın kullanım koşullarına uygun şekilde kullanın. Aşırı sorgulama yapmaktan kaçının.
