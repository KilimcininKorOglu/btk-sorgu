# BTK Site Sorgulama Aracı v2.0

Türkiye'de erişime engellenen web sitelerini [BTK Site Bilgileri Sorgu Sayfası](https://internet.btk.gov.tr/sitesorgu) üzerinden otomatik olarak sorgulayan Node.js aracı.

**✨ Özellikler:**

- 🤖 Google Gemini AI ile otomatik CAPTCHA çözümü
- 📋 Tek veya çoklu site sorgulama
- 📁 Dosyadan liste okuma
- 📊 JSON formatında temiz çıktı desteği
- 🔄 Otomatik yeniden deneme (3x)
- ⏱️ 30 saniye HTTP timeout
- 🔀 HTTP redirect desteği

---

## 📦 Kurulum

### Gereksinimler

- Node.js 16+
- Google Gemini API anahtarı (**zorunlu**)

### 1. Dosyaları İndir

```bash
git clone <repo-url>
cd btk-sorgu
```

### 2. Gemini API Anahtarı Al

1. [Google AI Studio](https://aistudio.google.com/app/apikey) adresine gidin
2. Google hesabınızla giriş yapın
3. **"Create API Key"** butonuna tıklayın
4. API anahtarını kopyalayın

### 3. Ortam Değişkenlerini Ayarla

**Seçenek 1: `.env` Dosyası (Önerilen)**

Proje klasöründe `.env` dosyası oluşturun:

```env
GEMINI_API_KEY=AIzaSy...your_api_key_here
GEMINI_MODEL=gemini-2.5-flash
```

> 💡 `.env.example` dosyasını `.env` olarak kopyalayabilirsiniz.

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

| Değişken | Zorunlu | Varsayılan | Açıklama |
|----------|---------|------------|----------|
| `GEMINI_API_KEY` | Evet | - | Google Gemini API anahtarı |
| `GEMINI_MODEL` | Hayır | `gemini-2.5-flash` | Kullanılacak Gemini modeli |
| `USER_AGENT` | Hayır | Chrome 120 | Özel User-Agent string |

---

## 🚀 Kullanım

### Temel Kullanım

```bash
# Tek site sorgula
node btk-sorgu.js discord.com
```

### Çoklu Site Sorgulama

```bash
# Birden fazla site
node btk-sorgu.js discord.com pornhub.com google.com twitter.com

# Dosyadan liste okuma
node btk-sorgu.js --liste sites.txt
```

### JSON Çıktı

```bash
# JSON formatında çıktı (sadece JSON, progress mesajı yok)
node btk-sorgu.js --json discord.com

# Dosyaya kaydet
node btk-sorgu.js --json discord.com > sonuc.json
```

### Versiyon ve Yardım

```bash
node btk-sorgu.js --version
node btk-sorgu.js --help
```

---

## 📋 Komut Satırı Seçenekleri

| Seçenek | Açıklama |
|---------|----------|
| `--liste <dosya>` | Dosyadan site listesi oku |
| `--json` | JSON formatında çıktı (temiz, progress yok) |
| `--version`, `-v` | Versiyon bilgisini göster |
| `--help`, `-h` | Yardım mesajını göster |

---

## 📁 Liste Dosyası Formatı

`sites.txt` örneği:

```text
# Yorum satırları # ile başlar
discord.com
pornhub.com
twitter.com
google.com
```

---

## 📊 Örnek Çıktılar

### Engellenmiş Site

```text
════════════════════════════════════════════════════════════
📌 Domain: discord.com
════════════════════════════════════════════════════════════
🚫 Durum: ENGELLİ
────────────────────────────────────────────────────────────
📅 Karar Tarihi: 09/10/2024
📋 Dosya Numarası: 2024/12907
📂 Dosya Türü: D. İş
⚖️ Mahkeme: Ankara 1. Sulh Ceza Hakimliği
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

```text
════════════════════════════════════════════════════════════
📌 Domain: google.com
════════════════════════════════════════════════════════════
✅ Durum: ERİŞİLEBİLİR
────────────────────────────────────────────────────────────
ℹ️ Bu site hakkında herhangi bir engel kararı bulunmamaktadır.
════════════════════════════════════════════════════════════
```

### JSON Çıktı (Başarılı)

```json
{
  "domain": "discord.com",
  "timestamp": "2024-11-27T10:30:00.000Z",
  "status": true,
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

### JSON Çıktı (Hata)

```json
{
  "domain": "example.com",
  "timestamp": "2024-11-27T10:30:00.000Z",
  "status": false,
  "error": "CAPTCHA çözümü başarısız oldu"
}
```

### Çoklu Sorgu Özeti

```text
📊 ÖZET
════════════════════════════════════════════════════════════
   🚫 Engelli: 2
   ✅ Erişilebilir: 1
════════════════════════════════════════════════════════════
```

---

## ⚙️ Yapılandırma

Script içindeki `CONFIG` objesi ile ayarları değiştirebilirsiniz:

```javascript
const CONFIG = {
  MAX_RETRIES: 3,           // CAPTCHA yeniden deneme sayısı
  RETRY_DELAY: 1000,        // Denemeler arası bekleme (ms)
  REQUEST_TIMEOUT: 30000,   // HTTP timeout (ms)
};
```

---

## 🔧 Sorun Giderme

### "GEMINI_API_KEY ayarlanmamış"

`.env` dosyası oluşturun veya ortam değişkeni ayarlayın.

### "CAPTCHA çözülemedi" / "MAX_TOKENS" hatası

- Gemini API anahtarınızın geçerli olduğundan emin olun
- `gemini-2.0-flash` veya `gemini-1.5-flash` modeli deneyin
- Script otomatik olarak 3 kez yeniden dener

### "Session başlatılamadı" hatası

- İnternet bağlantınızı kontrol edin
- BTK sunucusu geçici olarak erişilemez olabilir

### "İstek zaman aşımı" hatası

- Ağ bağlantınızı kontrol edin
- 30 saniye içinde yanıt alınamadı

---

## 📝 Teknik Detaylar

### Nasıl Çalışır?

1. **Session Başlatma:** BTK ana sayfasına GET isteği, session cookie'leri alınır
2. **CAPTCHA İndirme:** Session cookie'leri ile CAPTCHA resmi indirilir
3. **CAPTCHA Çözme:** Resim base64'e çevrilip Gemini API'ye gönderilir
4. **Sorgu Gönderme:** POST isteği ile site sorgulanır
5. **Sonuç Parse:** HTML yanıtından engel bilgileri regex ile çıkarılır

### API Endpoints

| Endpoint | Method | Açıklama |
|----------|--------|----------|
| `/sitesorgu/` | GET | Session cookie al |
| `/sitesorgu/secureimage/captcha.php` | GET | CAPTCHA resmi indir |
| `/sitesorgu/` | POST | Site sorgula |

### POST Parametreleri

| Parametre | Değer |
|-----------|-------|
| `deger` | Sorgulanacak domain |
| `security_code` | CAPTCHA kodu |
| `submit` | "Sorgula" |
| `ayrintili` | "0" |

---

## 💰 Maliyet

- **Gemini API:** Ücretsiz tier günde 60 istek/dakika destekler
- Her CAPTCHA çözümü = 1 API isteği
- Pratikte sınırsız kullanım

---

## ⚠️ Yasal Uyarı

Bu araç yalnızca **eğitim ve bilgilendirme** amaçlıdır. BTK'nın kullanım koşullarına uygun şekilde kullanın. Aşırı sorgulama yapmaktan kaçının.
