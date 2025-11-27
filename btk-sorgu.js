/**
 * BTK Site Sorgulama Script v2.0
 * ==============================
 * Türkiye'de engelli siteleri BTK üzerinden sorgular.
 * Gemini API ile CAPTCHA otomatik çözümü yapar.
 * 
 * Kullanım:
 *   node btk-sorgu.js <domain>                  Tek site sorgula
 *   node btk-sorgu.js --liste sites.txt         Liste ile sorgula
 *   node btk-sorgu.js --json <domain>           JSON formatında çıktı
 * 
 * Ortam Değişkenleri (.env dosyasından veya sistem ortamından):
 *   GEMINI_API_KEY    Google Gemini API anahtarı (ZORUNLU)
 *   GEMINI_MODEL      Gemini model adı (varsayılan: gemini-2.0-flash)
 * 
 * API Anahtarı Alma:
 *   https://aistudio.google.com/app/apikey
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const readline = require('readline');
const zlib = require('zlib');

// ============================================================================
// .ENV DOSYASI YÜKLEME (Zero-dependency)
// ============================================================================

/**
 * .env dosyasını okur ve ortam değişkenlerine yükler
 */
function loadEnvFile() {
  const envPath = path.join(process.cwd(), '.env');

  if (!fs.existsSync(envPath)) {
    return; // .env dosyası yoksa sessizce devam et
  }

  try {
    const content = fs.readFileSync(envPath, 'utf-8');
    const lines = content.split('\n');

    for (const line of lines) {
      // Boş satırları ve yorumları atla
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) {
        continue;
      }

      // KEY=VALUE formatını parse et
      const equalIndex = trimmed.indexOf('=');
      if (equalIndex === -1) {
        continue;
      }

      const key = trimmed.substring(0, equalIndex).trim();
      let value = trimmed.substring(equalIndex + 1).trim();

      // Tırnak işaretlerini kaldır
      if ((value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))) {
        value = value.slice(1, -1);
      }

      // Sadece tanımlı değilse ayarla (sistem ortam değişkenleri öncelikli)
      if (!process.env[key]) {
        process.env[key] = value;
      }
    }
  } catch (error) {
    console.error(`⚠️  .env dosyası okunamadı: ${error.message}`);
  }
}

// .env dosyasını yükle
loadEnvFile();

// ============================================================================
// YAPILANDIRMA
// ============================================================================

// Varsayılan Gemini model adı
const DEFAULT_GEMINI_MODEL = 'gemini-2.5-flash';

const CONFIG = {
  // BTK Ayarları
  BASE_URL: 'https://internet.btk.gov.tr/sitesorgu',
  CAPTCHA_PATH: '/secureimage/captcha.php',
  HEADERS: {
    'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8',
    'Accept-Language': 'tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7',
    'Accept-Encoding': 'gzip, deflate, br',
    'Origin': 'https://internet.btk.gov.tr',
    'Referer': 'https://internet.btk.gov.tr/sitesorgu/',
    'Connection': 'keep-alive',
    'Cache-Control': 'no-cache',
  },
  CAPTCHA_FILE: 'captcha.png',

  // Gemini API Ayarları (.env dosyasından veya varsayılan)
  get GEMINI_MODEL() {
    return process.env.GEMINI_MODEL || DEFAULT_GEMINI_MODEL;
  },
  get GEMINI_API_URL() {
    return `https://generativelanguage.googleapis.com/v1beta/models/${this.GEMINI_MODEL}:generateContent`;
  },
  GEMINI_PROMPT: `Bu bir CAPTCHA görüntüsüdür. Görüntüdeki karakterleri aynen oku.

ÖNEMLİ KURALLAR:
- SADECE gördüğün karakterleri yaz, başka hiçbir şey yazma
- Büyük/küçük harf AYNEN olmalı (case-sensitive)
- 5 veya 6 karakter olacak
- Örnek: zQsmR veya A8kN2P

Şimdi resimdeki kodu yaz:`,

  // Yeniden deneme ayarları
  MAX_RETRIES: 3,
  RETRY_DELAY: 1000,
};

// ============================================================================
// YARDIMCI FONKSİYONLAR
// ============================================================================

/**
 * Unix timestamp ile microseconds formatı oluşturur
 * Format: "0.XXXXXXXX UNIXTIME"
 */
function generateTimestamp() {
  const now = Date.now();
  const seconds = Math.floor(now / 1000);
  const microseconds = (now % 1000) / 1000;
  return `${microseconds.toFixed(8)} ${seconds}`;
}

/**
 * Cookie'leri parse eder
 */
function parseCookies(setCookieHeaders) {
  if (!setCookieHeaders) return {};
  const cookies = {};
  const cookieArray = Array.isArray(setCookieHeaders) ? setCookieHeaders : [setCookieHeaders];

  cookieArray.forEach(cookie => {
    const parts = cookie.split(';')[0].split('=');
    if (parts.length >= 2) {
      cookies[parts[0].trim()] = parts.slice(1).join('=').trim();
    }
  });

  return cookies;
}

/**
 * Cookie objesini string'e çevirir
 */
function cookiesToString(cookies) {
  return Object.entries(cookies)
    .map(([key, value]) => `${key}=${value}`)
    .join('; ');
}

/**
 * Basit HTML parser - cheerio olmadan
 */
function parseHTML(html) {
  const result = {
    turkceAciklama: null,
    ingilizceAciklama: null,
    engelliMi: false,
    kararTarihi: null,
    kararNumarasi: null,
    dosyaNumarasi: null,
    dosyaTuru: null,
    mahkeme: null,
  };

  // Türkçe açıklama (.yazi2_2)
  const turkceMatch = html.match(/<span class="yazi2_2">([\s\S]*?)<\/span>/i);
  if (turkceMatch) {
    result.turkceAciklama = turkceMatch[1]
      .replace(/<[^>]*>/g, '')
      .replace(/&nbsp;/g, ' ')
      .trim();
  }

  // İngilizce açıklama (.yazi3_1)
  const ingilizceMatch = html.match(/<span class="yazi3_1">([\s\S]*?)<\/span>/i);
  if (ingilizceMatch) {
    result.ingilizceAciklama = ingilizceMatch[1]
      .replace(/<[^>]*>/g, '')
      .replace(/&nbsp;/g, ' ')
      .trim();
  }

  // Engel durumu kontrolü
  if (result.turkceAciklama) {
    result.engelliMi = result.turkceAciklama.includes('engellenmiştir');

    // Karar bilgilerini çıkar
    // Desteklenen dosya türleri: D. İş, E., K., Müt., vb.
    const kararMatch = result.turkceAciklama.match(
      /(\d{2}\/\d{2}\/\d{4}) tarihli ve ((\d+\/\d+)\s+([A-Za-zİıÜüÖöÇçŞşĞğ.\s]+?)) sayılı (.+?) kararıyla/
    );

    if (kararMatch) {
      result.kararTarihi = kararMatch[1];
      result.kararNumarasi = kararMatch[2].trim();
      result.dosyaNumarasi = kararMatch[3];
      result.dosyaTuru = kararMatch[4].trim();
      result.mahkeme = kararMatch[5];
    }
  }

  // Engel yok mesajı kontrolü - farklı formatlar
  const noBlockPatterns = [
    /herhangi bir (idari|yargı) karar[ıi] bulunmamaktadır/i,
    /uygulanan bir karar bulunamadı/i,
    /karar bulunamadı/i,
    /engel.{0,20}bulunmamaktadır/i
  ];

  for (const pattern of noBlockPatterns) {
    if (pattern.test(html)) {
      result.engelliMi = false;
      result.turkceAciklama = 'Bu site hakkında herhangi bir engel kararı bulunmamaktadır.';
      break;
    }
  }

  return result;
}

/**
 * Bekleme fonksiyonu
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// ============================================================================
// HTTP İSTEK FONKSİYONLARI
// ============================================================================

/**
 * Sıkıştırılmış veriyi açar
 */
function decompressResponse(buffer, encoding) {
  return new Promise((resolve, reject) => {
    if (!encoding) {
      resolve(buffer);
      return;
    }

    if (encoding === 'gzip') {
      zlib.gunzip(buffer, (err, result) => {
        if (err) reject(err);
        else resolve(result);
      });
    } else if (encoding === 'deflate') {
      zlib.inflate(buffer, (err, result) => {
        if (err) reject(err);
        else resolve(result);
      });
    } else if (encoding === 'br') {
      zlib.brotliDecompress(buffer, (err, result) => {
        if (err) reject(err);
        else resolve(result);
      });
    } else {
      resolve(buffer);
    }
  });
}

/**
 * HTTPS GET isteği yapar
 */
function httpsGet(url, options = {}) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);

    // Accept-Encoding header'ını ayarla (gzip ve deflate destekle, br hariç)
    const headers = { ...CONFIG.HEADERS, ...options.headers };
    headers['Accept-Encoding'] = 'gzip, deflate';

    const reqOptions = {
      hostname: urlObj.hostname,
      port: 443,
      path: urlObj.pathname + urlObj.search,
      method: 'GET',
      headers: headers,
    };

    const req = https.request(reqOptions, (res) => {
      const chunks = [];

      res.on('data', chunk => chunks.push(chunk));
      res.on('end', async () => {
        try {
          const rawData = Buffer.concat(chunks);
          const encoding = res.headers['content-encoding'];
          const data = await decompressResponse(rawData, encoding);

          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            data: data,
          });
        } catch (err) {
          reject(err);
        }
      });
    });

    req.on('error', reject);
    req.end();
  });
}

/**
 * HTTPS POST isteği yapar (form data)
 */
function httpsPost(url, body, options = {}) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const postData = typeof body === 'string' ? body : new URLSearchParams(body).toString();

    const reqOptions = {
      hostname: urlObj.hostname,
      port: 443,
      path: urlObj.pathname + urlObj.search,
      method: 'POST',
      headers: {
        ...CONFIG.HEADERS,
        'Content-Type': 'application/x-www-form-urlencoded',
        'Content-Length': Buffer.byteLength(postData),
        ...options.headers,
      },
    };

    const req = https.request(reqOptions, (res) => {
      const chunks = [];

      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          data: Buffer.concat(chunks).toString('utf-8'),
        });
      });
    });

    req.on('error', reject);
    req.write(postData);
    req.end();
  });
}

/**
 * HTTPS POST isteği yapar (JSON data)
 */
function httpsPostJSON(url, jsonBody, options = {}) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const postData = JSON.stringify(jsonBody);

    const reqOptions = {
      hostname: urlObj.hostname,
      port: 443,
      path: urlObj.pathname + urlObj.search,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
        ...options.headers,
      },
    };

    const req = https.request(reqOptions, (res) => {
      const chunks = [];

      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        const responseData = Buffer.concat(chunks).toString('utf-8');
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          data: responseData,
        });
      });
    });

    req.on('error', reject);
    req.write(postData);
    req.end();
  });
}

// ============================================================================
// GEMINI API FONKSİYONLARI
// ============================================================================

/**
 * Gemini API ile CAPTCHA çözer
 * @param {Buffer} imageBuffer - CAPTCHA resmi buffer'ı
 * @param {string} apiKey - Gemini API anahtarı
 * @returns {Promise<string>} - Çözülmüş CAPTCHA kodu
 */
async function solveCaptchaWithGemini(imageBuffer, apiKey) {
  console.log('🤖 Gemini API ile CAPTCHA çözülüyor...');

  // Base64'e çevir
  const base64Image = imageBuffer.toString('base64');

  // Gemini API isteği oluştur
  const requestBody = {
    contents: [
      {
        parts: [
          {
            text: CONFIG.GEMINI_PROMPT
          },
          {
            inline_data: {
              mime_type: 'image/png',
              data: base64Image
            }
          }
        ]
      }
    ],
    generationConfig: {
      temperature: 0.1,
      topK: 1,
      topP: 1,
      maxOutputTokens: 10,
    }
  };

  const url = CONFIG.GEMINI_API_URL;

  try {
    const response = await httpsPostJSON(url, requestBody, {
      headers: {
        'x-goog-api-key': apiKey
      }
    });

    if (response.statusCode !== 200) {
      const errorData = JSON.parse(response.data);
      throw new Error(`Gemini API hatası: ${errorData.error?.message || response.statusCode}`);
    }

    const data = JSON.parse(response.data);

    // Yanıtı parse et
    const text = data.candidates?.[0]?.content?.parts?.[0]?.text;

    if (!text) {
      throw new Error('Gemini API yanıt vermedi');
    }

    // Sadece alfanumerik karakterleri al (5-6 karakter) - CASE SENSITIVE!
    const captchaCode = text.replace(/[^A-Za-z0-9]/g, '');

    if (captchaCode.length < 5 || captchaCode.length > 6) {
      throw new Error(`Geçersiz CAPTCHA çıktısı: "${text}" -> "${captchaCode}" (${captchaCode.length} karakter)`);
    }

    console.log(`✅ CAPTCHA çözüldü: ${captchaCode}`);
    return captchaCode;

  } catch (error) {
    if (error.message.includes('API')) {
      throw error;
    }
    throw new Error(`Gemini API isteği başarısız: ${error.message}`);
  }
}

// ============================================================================
// BTK FONKSİYONLARI
// ============================================================================

/**
 * Ana sayfadan session cookie alır
 */
async function getSessionCookies() {
  console.log('🔗 Session başlatılıyor...');

  const response = await httpsGet(`${CONFIG.BASE_URL}/`);

  if (response.statusCode !== 200) {
    throw new Error(`Session başlatılamadı: HTTP ${response.statusCode}`);
  }

  const cookies = parseCookies(response.headers['set-cookie']);
  console.log(`✅ Session alındı: ${Object.keys(cookies).length} cookie`);

  return cookies;
}

/**
 * CAPTCHA resmini indirir
 * @returns {Promise<{cookies: Object, imageBuffer: Buffer, captchaPath: string}>}
 */
async function getCaptcha() {
  // Önce session cookie al
  const sessionCookies = await getSessionCookies();

  const timestamp = generateTimestamp();
  const url = `${CONFIG.BASE_URL}${CONFIG.CAPTCHA_PATH}?_CAPTCHA=&t=${encodeURIComponent(timestamp)}`;

  console.log('📥 CAPTCHA indiriliyor...');

  const response = await httpsGet(url, {
    headers: {
      Cookie: cookiesToString(sessionCookies),
      'Accept': 'image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8',
    }
  });

  if (response.statusCode !== 200) {
    throw new Error(`CAPTCHA indirilemedi: HTTP ${response.statusCode}`);
  }

  // Cookie'leri birleştir
  const newCookies = parseCookies(response.headers['set-cookie']);
  const cookies = { ...sessionCookies, ...newCookies };

  // Veri kontrolü
  if (!response.data || response.data.length === 0) {
    throw new Error('CAPTCHA resmi boş döndü! BTK sunucusu yanıt vermedi.');
  }

  // CAPTCHA resmini kaydet
  const captchaPath = path.join(process.cwd(), CONFIG.CAPTCHA_FILE);
  fs.writeFileSync(captchaPath, response.data);

  console.log(`✅ CAPTCHA kaydedildi: ${captchaPath} (${response.data.length} bytes)`);

  return {
    cookies,
    imageBuffer: response.data,
    captchaPath
  };
}

/**
 * Site sorgulama isteği gönderir
 */
async function sorgulaSite(domain, captchaCode, cookies) {
  console.log(`\n🔍 Sorgulanıyor: ${domain}`);

  const formData = {
    deger: domain,
    ipw: '',
    kat: '',
    tr: '',
    eg: '',
    ayrintili: '0',
    submit: 'Sorgula',
    security_code: captchaCode,
  };

  const response = await httpsPost(`${CONFIG.BASE_URL}/`, formData, {
    headers: {
      Cookie: cookiesToString(cookies),
    },
  });

  if (response.statusCode !== 200) {
    throw new Error(`Sorgu başarısız: HTTP ${response.statusCode}`);
  }

  return response.data;
}

/**
 * CAPTCHA hatalı mı kontrol eder
 */
function isCaptchaError(html) {
  return html.includes('Güvenlik kodu hatalı') ||
    html.includes('security code') ||
    html.includes('Doğrulama kodu');
}

/**
 * Sonuçları güzel formatta yazdırır
 */
function printResult(domain, result) {
  console.log('\n' + '═'.repeat(60));
  console.log(`📌 Domain: ${domain}`);
  console.log('═'.repeat(60));

  if (result.engelliMi) {
    console.log('🚫 Durum: ENGELLİ');
    console.log('─'.repeat(60));

    if (result.kararTarihi) {
      console.log(`📅 Karar Tarihi: ${result.kararTarihi}`);
    }
    if (result.dosyaNumarasi) {
      console.log(`📋 Dosya Numarası: ${result.dosyaNumarasi}`);
    }
    if (result.dosyaTuru) {
      console.log(`📂 Dosya Türü: ${result.dosyaTuru}`);
    }
    if (result.mahkeme) {
      console.log(`⚖️ Mahkeme: ${result.mahkeme}`);
    }

    console.log('─'.repeat(60));

    if (result.turkceAciklama) {
      console.log('\n📝 Türkçe Açıklama:');
      console.log(`   ${result.turkceAciklama}`);
    }

    if (result.ingilizceAciklama) {
      console.log('\n📝 English Description:');
      console.log(`   ${result.ingilizceAciklama}`);
    }
  } else {
    console.log('✅ Durum: ERİŞİLEBİLİR');
    console.log('─'.repeat(60));
    console.log('ℹ️ Bu site hakkında herhangi bir engel kararı bulunmamaktadır.');
  }

  console.log('═'.repeat(60) + '\n');

  return result;
}

/**
 * JSON formatında çıktı verir
 */
function outputJSON(domain, result) {
  const output = {
    domain,
    timestamp: new Date().toISOString(),
    ...result,
  };

  console.log(JSON.stringify(output, null, 2));
  return output;
}

/**
 * Yardım mesajını gösterir
 */
function showHelp() {
  console.log(`
╔════════════════════════════════════════════════════════════╗
║           BTK Site Sorgulama Aracı v2.0                    ║
║           https://internet.btk.gov.tr/sitesorgu            ║
╚════════════════════════════════════════════════════════════╝

Kullanım:
  node btk-sorgu.js [seçenekler] <domain>

Seçenekler:
  --liste <dosya>     Dosyadan site listesi oku
  --json              JSON formatında çıktı
  --help, -h          Bu yardım mesajını göster

Örnekler:
  node btk-sorgu.js discord.com
  node btk-sorgu.js discord.com twitter.com google.com
  node btk-sorgu.js --liste sites.txt
  node btk-sorgu.js --json twitter.com

Ortam Değişkenleri (.env dosyası veya sistem ortamı):
  GEMINI_API_KEY      Google Gemini API anahtarı (ZORUNLU)
  GEMINI_MODEL        Gemini model adı (varsayılan: gemini-2.5-flash)

.env Dosyası Örneği:
  GEMINI_API_KEY=AIzaSy...your_api_key_here
  GEMINI_MODEL=gemini-2.5-flash

API Anahtarı Alma:
  https://aistudio.google.com/app/apikey
`);
}

// ============================================================================
// ANA PROGRAM
// ============================================================================

async function main() {
  // Komut satırı argümanlarını parse et
  const args = process.argv.slice(2);

  // Yardım kontrolü
  if (args.includes('--help') || args.includes('-h') || args.length === 0) {
    showHelp();
    process.exit(args.length === 0 ? 1 : 0);
  }

  console.log(`
╔════════════════════════════════════════════════════════════╗
║           BTK Site Sorgulama Aracı v2.0                    ║
║           https://internet.btk.gov.tr/sitesorgu            ║
╚════════════════════════════════════════════════════════════╝
`);

  let domains = [];
  let jsonOutput = false;

  // Argümanları işle
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--liste' && args[i + 1]) {
      const listFile = args[i + 1];
      if (!fs.existsSync(listFile)) {
        console.error(`❌ Dosya bulunamadı: ${listFile}`);
        process.exit(1);
      }
      const content = fs.readFileSync(listFile, 'utf-8');
      domains = content.split('\n')
        .map(line => line.trim())
        .filter(line => line && !line.startsWith('#'));
      i++;
    } else if (args[i] === '--json') {
      jsonOutput = true;
    } else if (!args[i].startsWith('--')) {
      domains.push(args[i]);
    }
  }

  if (domains.length === 0) {
    console.error('❌ Sorgulanacak domain belirtilmedi!');
    console.log('   Kullanım: node btk-sorgu.js <domain>');
    process.exit(1);
  }

  // Gemini API key kontrolü (ZORUNLU)
  const geminiApiKey = process.env.GEMINI_API_KEY;
  if (!geminiApiKey) {
    console.error('❌ GEMINI_API_KEY ayarlanmamış!');
    console.log('');
    console.log('   Seçenek 1: .env dosyası oluşturun');
    console.log('   GEMINI_API_KEY=your_api_key');
    console.log('');
    console.log('   Seçenek 2: Ortam değişkeni ayarlayın');
    console.log('   Windows: set GEMINI_API_KEY=your_api_key');
    console.log('   Linux/Mac: export GEMINI_API_KEY=your_api_key');
    console.log('');
    console.log('   API anahtarı almak için: https://aistudio.google.com/app/apikey');
    process.exit(1);
  }

  console.log(`📋 Sorgulanacak ${domains.length} site: ${domains.join(', ')}`);
  console.log(`🤖 Model: ${CONFIG.GEMINI_MODEL}\n`);

  const results = [];
  let retryCount = 0;

  try {
    while (retryCount < CONFIG.MAX_RETRIES) {
      // 1. CAPTCHA al
      const { cookies, imageBuffer, captchaPath } = await getCaptcha();

      let captchaCode;

      // Gemini ile otomatik çöz
      try {
        captchaCode = await solveCaptchaWithGemini(imageBuffer, geminiApiKey);
      } catch (error) {
        console.error(`❌ CAPTCHA çözülemedi: ${error.message}`);
        retryCount++;
        if (retryCount < CONFIG.MAX_RETRIES) {
          console.log(`🔄 Yeniden deneniyor (${retryCount}/${CONFIG.MAX_RETRIES})...`);
          await sleep(CONFIG.RETRY_DELAY);
          continue;
        }
        throw error;
      }

      // 3. İlk siteyi sorgula (CAPTCHA doğrulama)
      const firstDomain = domains[0];
      const firstHtml = await sorgulaSite(firstDomain, captchaCode, cookies);

      // CAPTCHA hatalı mı kontrol et
      if (isCaptchaError(firstHtml)) {
        console.log('⚠️  CAPTCHA kodu hatalı!');
        retryCount++;
        if (retryCount < CONFIG.MAX_RETRIES) {
          console.log(`🔄 Yeni CAPTCHA ile deneniyor (${retryCount}/${CONFIG.MAX_RETRIES})...`);
          await sleep(CONFIG.RETRY_DELAY);
          continue;
        }
        throw new Error('CAPTCHA çözümü başarısız oldu');
      }

      // İlk sonucu işle
      const firstResult = parseHTML(firstHtml);
      if (jsonOutput) {
        results.push(outputJSON(firstDomain, firstResult));
      } else {
        results.push(printResult(firstDomain, firstResult));
      }

      // Başarılı - döngüden çık
      break;
    }

    // 4. Kalan siteleri sorgula (her biri için yeni CAPTCHA gerekiyor)
    for (let i = 1; i < domains.length; i++) {
      const domain = domains[i];
      let domainRetry = 0;

      while (domainRetry < CONFIG.MAX_RETRIES) {
        try {
          // Her site için yeni session ve CAPTCHA al
          const { cookies: newCookies, imageBuffer: newImage } = await getCaptcha();

          const newCaptchaCode = await solveCaptchaWithGemini(newImage, geminiApiKey);

          const html = await sorgulaSite(domain, newCaptchaCode, newCookies);

          // CAPTCHA hatalı mı?
          if (isCaptchaError(html)) {
            domainRetry++;
            if (domainRetry < CONFIG.MAX_RETRIES) {
              console.log(`⚠️  CAPTCHA hatalı, yeniden deneniyor (${domainRetry}/${CONFIG.MAX_RETRIES})...`);
              await sleep(CONFIG.RETRY_DELAY);
              continue;
            }
            throw new Error('CAPTCHA çözümü başarısız');
          }

          const result = parseHTML(html);

          if (jsonOutput) {
            results.push(outputJSON(domain, result));
          } else {
            results.push(printResult(domain, result));
          }

          break; // Bu domain için başarılı

        } catch (error) {
          domainRetry++;
          if (domainRetry >= CONFIG.MAX_RETRIES) {
            console.error(`❌ ${domain} sorgulanırken hata: ${error.message}`);
          } else {
            console.log(`🔄 ${domain} için yeniden deneniyor...`);
            await sleep(CONFIG.RETRY_DELAY);
          }
        }
      }

      // Rate limiting
      if (i < domains.length - 1) {
        await sleep(500);
      }
    }

    // 5. Sonuç özeti
    if (!jsonOutput && domains.length > 1) {
      console.log('\n📊 ÖZET');
      console.log('═'.repeat(60));

      const blocked = results.filter(r => r?.engelliMi).length;
      const accessible = results.filter(r => r && !r.engelliMi).length;
      const failed = domains.length - results.length;

      console.log(`   🚫 Engelli: ${blocked}`);
      console.log(`   ✅ Erişilebilir: ${accessible}`);
      if (failed > 0) {
        console.log(`   ❓ Hatalı: ${failed}`);
      }
      console.log('═'.repeat(60));
    }

    // CAPTCHA dosyasını temizle
    const captchaPath = path.join(process.cwd(), CONFIG.CAPTCHA_FILE);
    if (fs.existsSync(captchaPath)) {
      fs.unlinkSync(captchaPath);
      if (!jsonOutput) {
        console.log('\n🧹 CAPTCHA dosyası temizlendi.');
      }
    }

  } catch (error) {
    console.error(`\n❌ Hata: ${error.message}`);
    process.exit(1);
  }
}

// Programı çalıştır
main().catch(console.error);
