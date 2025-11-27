# BTK Site Sorgulama Aracı v2.0

Türkiye'de erişime engellenen web sitelerini [BTK Site Bilgileri Sorgu Sayfası](https://internet.btk.gov.tr/sitesorgu) üzerinden otomatik olarak sorgulayan Node.js aracı.

**✨ Özellikler:**

- 🤖 Google Gemini AI ile otomatik CAPTCHA çözümü
- 📋 Tek veya çoklu site sorgulama
- 📁 Dosyadan liste okuma
- 📊 JSON formatında çıktı desteği
- 🔄 Otomatik yeniden deneme (3x)

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

### 3. Ortam Değişkenini Ayarla

**Windows (CMD):**

```cmd
set GEMINI_API_KEY=AIzaSy...your_api_key_here
```

**Windows (PowerShell):**

```powershell
$env:GEMINI_API_KEY="AIzaSy...your_api_key_here"
```

**Linux/macOS:**

```bash
export GEMINI_API_KEY=AIzaSy...your_api_key_here
```

**Kalıcı Ayar (Windows):**

```cmd
setx GEMINI_API_KEY "AIzaSy...your_api_key_here"
```

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
# JSON formatında çıktı
node btk-sorgu.js --json discord.com

# Dosyaya kaydet
node btk-sorgu.js --json discord.com > sonuc.json
```

### Yardım

```bash
node btk-sorgu.js --help
```

---

## 📋 Komut Satırı Seçenekleri

| Seçenek | Açıklama |
|---------|----------|
| `--liste <dosya>` | Dosyadan site listesi oku |
| `--json` | JSON formatında çıktı |
| `--help`, `-h` | Yardım mesajını göster |

---

## 📁 Liste Dosyası Formatı

`sites.txt` örneği:

```
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
📋 Karar Numarası: 2024/12907 D. İş
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

```text
════════════════════════════════════════════════════════════
📌 Domain: google.com
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
  "timestamp": "2024-11-26T13:45:00.000Z",
  "engelliMi": true,
  "kararTarihi": "09/10/2024",
  "kararNumarasi": "2024/12907 D. İş",
  "mahkeme": "Ankara 1. Sulh Ceza Hakimliği",
  "turkceAciklama": "discord.com, 09/10/2024 tarihli ve 2024/12907 D. İş sayılı Ankara 1. Sulh Ceza Hakimliği kararıyla erişime engellenmiştir.",
  "ingilizceAciklama": "discord.com has been blocked by the decision dated 09/10/2024 and numbered 2024/12907 D. İş of Ankara 1. Sulh Ceza Hakimliği."
}
```

### Çoklu Sorgu Özeti

```
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
  // Yeniden deneme sayısı (CAPTCHA hatalı olursa)
  MAX_RETRIES: 3,
  
  // Yeniden denemeler arası bekleme (ms)
  RETRY_DELAY: 1000,
  
  // Gemini modeli
  GEMINI_API_URL: 'https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent',
};
```

---

## 🔧 Sorun Giderme

### "GEMINI_API_KEY ortam değişkeni ayarlanmamış"

Ortam değişkenini ayarlayın:

```bash
# Windows
set GEMINI_API_KEY=your_api_key

# Linux/Mac
export GEMINI_API_KEY=your_api_key
```

### "CAPTCHA çözülemedi" hatası

- Gemini API anahtarınızın geçerli olduğundan emin olun
- API kotanızı kontrol edin (günlük limit)
- Script otomatik olarak 3 kez yeniden dener

### "Session başlatılamadı" hatası

- İnternet bağlantınızı kontrol edin
- BTK sunucusu geçici olarak erişilemez olabilir
- Bir süre bekleyip tekrar deneyin

### CAPTCHA sürekli hatalı

- Gemini bazen CAPTCHA'yı yanlış okuyabilir
- Script otomatik olarak 3 kez yeniden dener

---

## 📝 Teknik Detaylar

### Nasıl Çalışır?

1. **Session Başlatma:** BTK ana sayfasına GET isteği yapılır, session cookie'leri alınır
2. **CAPTCHA İndirme:** Session cookie'leri ile CAPTCHA resmi indirilir
3. **CAPTCHA Çözme:** Resim base64'e çevrilip Gemini API'ye gönderilir
4. **Sorgu Gönderme:** POST isteği ile site sorgulanır
5. **Sonuç Parse:** HTML yanıtından engel bilgileri çıkarılır

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
