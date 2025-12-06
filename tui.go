package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// History dosya adı
const historyFileName = "history.json"

// HistoryData geçmiş sorgu verileri
type HistoryData struct {
	Queries []QueryResult `json:"queries"`
}

// loadHistory geçmiş sorguları dosyadan yükler
func loadHistory() []QueryResult {
	historyPath := filepath.Join(".", historyFileName)
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return []QueryResult{}
	}

	var history HistoryData
	if err := json.Unmarshal(data, &history); err != nil {
		return []QueryResult{}
	}

	return history.Queries
}

// saveHistory sorguları dosyaya kaydeder
func saveHistory(results []QueryResult) error {
	historyPath := filepath.Join(".", historyFileName)
	history := HistoryData{Queries: results}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyPath, data, 0644)
}

// TUI stilleri
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	blockedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	accessibleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)
)

// TUI durumları
type tuiState int

const (
	stateInput tuiState = iota
	stateQuerying
	stateResult
	stateHistory // Geçmiş tablosu aktif
)

// TUI model
type tuiModel struct {
	state          tuiState
	textInput      textinput.Model
	spinner        spinner.Model
	table          table.Model
	results        []QueryResult
	currentMsg     string
	err            error
	width          int
	height         int
	apiKey         string
	queryDomain    string
	refreshingIdx  int  // Güncellenen sorgunun index'i (-1 = yeni sorgu)
	inputFocused   bool // Input mu yoksa tablo mu odaklı
}

// Mesaj tipleri
type queryStartMsg struct {
	domain string
}

type queryProgressMsg struct {
	message string
}

type queryResultMsg struct {
	result QueryResult
}

type queryErrorMsg struct {
	err error
}

// TUI model oluştur
func newTUIModel(apiKey string) tuiModel {
	ti := textinput.New()
	ti.Placeholder = "discord.com"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	columns := []table.Column{
		{Title: "Domain", Width: 25},
		{Title: "Durum", Width: 15},
		{Title: "Süre", Width: 10},
		{Title: "Mahkeme", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(ts)

	// Geçmiş sorguları yükle
	history := loadHistory()

	model := tuiModel{
		state:         stateInput,
		textInput:     ti,
		spinner:       s,
		table:         t,
		results:       history,
		apiKey:        apiKey,
		refreshingIdx: -1,
		inputFocused:  true,
	}

	// Tablo'yu geçmiş verilerle güncelle
	model.updateTable()

	return model
}

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state != stateQuerying {
				return m, tea.Quit
			}
		case "ctrl+d":
			// Geçmişi temizle
			if (m.state == stateInput || m.state == stateHistory) && len(m.results) > 0 {
				m.results = []QueryResult{}
				m.updateTable()
				saveHistory(m.results)
				m.state = stateInput
				m.inputFocused = true
			}
		case "tab":
			// Input ve tablo arasında geçiş
			if m.state == stateInput && len(m.results) > 0 {
				m.state = stateHistory
				m.inputFocused = false
				m.textInput.Blur()
			} else if m.state == stateHistory {
				m.state = stateInput
				m.inputFocused = true
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "enter":
			if m.state == stateInput && m.textInput.Value() != "" {
				// Yeni sorgu
				domain := strings.TrimSpace(m.textInput.Value())
				if isValidDomain(domain) {
					m.state = stateQuerying
					m.queryDomain = domain
					m.refreshingIdx = -1 // Yeni sorgu
					m.currentMsg = "Session başlatılıyor..."
					return m, tea.Batch(m.spinner.Tick, m.startQuery(domain))
				} else {
					m.err = fmt.Errorf("geçersiz domain: %s", domain)
				}
			} else if m.state == stateHistory && len(m.results) > 0 {
				// Geçmişten seçilen domain'i tekrar sorgula
				selectedIdx := m.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(m.results) {
					domain := m.results[selectedIdx].Domain
					m.state = stateQuerying
					m.queryDomain = domain
					m.refreshingIdx = selectedIdx // Güncellenecek index
					m.currentMsg = "Yeniden sorgulanıyor..."
					return m, tea.Batch(m.spinner.Tick, m.startQuery(domain))
				}
			} else if m.state == stateResult {
				// Yeni sorgu için input'a dön
				m.state = stateInput
				m.textInput.SetValue("")
				m.textInput.Focus()
				m.inputFocused = true
				m.err = nil
				return m, textinput.Blink
			}
		case "esc":
			if m.state == stateResult {
				m.state = stateInput
				m.textInput.Focus()
				m.inputFocused = true
				return m, textinput.Blink
			} else if m.state == stateHistory {
				m.state = stateInput
				m.textInput.Focus()
				m.inputFocused = true
				return m, textinput.Blink
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		if m.state == stateQuerying {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case queryProgressMsg:
		m.currentMsg = msg.message

	case queryResultMsg:
		m.state = stateResult
		if m.refreshingIdx >= 0 && m.refreshingIdx < len(m.results) {
			// Mevcut kaydı güncelle
			m.results[m.refreshingIdx] = msg.result
		} else {
			// Yeni kayıt ekle
			m.results = append(m.results, msg.result)
		}
		m.refreshingIdx = -1
		m.updateTable()
		// Geçmişi kaydet
		saveHistory(m.results)

	case queryErrorMsg:
		m.state = stateResult
		m.err = msg.err
	}

	// Input güncellemesi
	if m.state == stateInput {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Tablo güncellemesi (history veya result modunda)
	if m.state == stateHistory || m.state == stateResult {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *tuiModel) updateTable() {
	var rows []table.Row
	for _, r := range m.results {
		status := accessibleStyle.Render("✅ Erişilebilir")
		if r.EngelliMi {
			status = blockedStyle.Render("🚫 Engelli")
		}
		if !r.Status {
			status = errorStyle.Render("❌ Hata")
		}

		mahkeme := r.Mahkeme
		if mahkeme == "" {
			mahkeme = "-"
		}
		if len(mahkeme) > 28 {
			mahkeme = mahkeme[:28] + "..."
		}

		rows = append(rows, table.Row{
			r.Domain,
			status,
			r.QueryDurationFormatted,
			mahkeme,
		})
	}
	m.table.SetRows(rows)
}

func (m tuiModel) startQuery(domain string) tea.Cmd {
	return func() tea.Msg {
		// HTTP client oluştur
		client = createHTTPClient()

		result := querySingleDomain(domain, m.apiKey)
		if !result.Status {
			return queryErrorMsg{err: fmt.Errorf(result.Error)}
		}
		return queryResultMsg{result: result}
	}
}

func (m tuiModel) View() string {
	var s strings.Builder

	// Başlık
	title := titleStyle.Render(" 🔍 BTK Site Sorgulama Aracı v" + Version + " ")
	s.WriteString(title + "\n\n")

	switch m.state {
	case stateInput:
		s.WriteString("Domain girin:\n\n")
		s.WriteString(inputStyle.Render(m.textInput.View()) + "\n")

		if m.err != nil {
			s.WriteString("\n" + errorStyle.Render("❌ "+m.err.Error()) + "\n")
		}

		if len(m.results) > 0 {
			s.WriteString("\n📊 Önceki Sorgular " + infoStyle.Render("(Tab ile seç)") + ":\n\n")
			s.WriteString(m.table.View() + "\n")
		}

		if len(m.results) > 0 {
			s.WriteString(helpStyle.Render("\n[Enter] Sorgula • [Tab] Geçmişe Git • [Ctrl+D] Temizle • [Q] Çıkış"))
		} else {
			s.WriteString(helpStyle.Render("\n[Enter] Sorgula • [Q] Çıkış"))
		}

	case stateHistory:
		s.WriteString("Domain girin:\n\n")
		s.WriteString(infoStyle.Render(m.textInput.View()) + "\n")

		s.WriteString("\n📊 Geçmiş Sorgular " + successStyle.Render("(↑↓ ile seç, Enter ile yenile)") + ":\n\n")
		s.WriteString(m.table.View() + "\n")

		s.WriteString(helpStyle.Render("\n[Enter] Seçili Sorguyu Yenile • [Tab/Esc] Geri • [Ctrl+D] Temizle • [Q] Çıkış"))

	case stateQuerying:
		s.WriteString(m.spinner.View() + " " + m.currentMsg + "\n")
		s.WriteString(helpStyle.Render("\nSorgulanıyor: " + m.queryDomain))

	case stateResult:
		if m.err != nil {
			s.WriteString(errorStyle.Render("❌ Hata: "+m.err.Error()) + "\n")
		} else if len(m.results) > 0 {
			lastResult := m.results[len(m.results)-1]

			// Son sonuç detayları
			var detail strings.Builder
			detail.WriteString(fmt.Sprintf("📌 Domain: %s\n", lastResult.Domain))
			detail.WriteString(fmt.Sprintf("⏱️  Süre: %s\n\n", lastResult.QueryDurationFormatted))

			if lastResult.EngelliMi {
				detail.WriteString(blockedStyle.Render("🚫 DURUM: ENGELLİ") + "\n\n")
				if lastResult.KararTarihi != "" {
					detail.WriteString(fmt.Sprintf("📅 Karar Tarihi: %s\n", lastResult.KararTarihi))
				}
				if lastResult.DosyaNumarasi != "" {
					detail.WriteString(fmt.Sprintf("📋 Dosya No: %s\n", lastResult.DosyaNumarasi))
				}
				if lastResult.Mahkeme != "" {
					detail.WriteString(fmt.Sprintf("⚖️  Mahkeme: %s\n", lastResult.Mahkeme))
				}
			} else {
				detail.WriteString(accessibleStyle.Render("✅ DURUM: ERİŞİLEBİLİR") + "\n\n")
				detail.WriteString("Bu site hakkında engel kararı bulunmamaktadır.\n")
			}

			s.WriteString(boxStyle.Render(detail.String()) + "\n")

			// Tüm sonuçlar tablosu
			if len(m.results) > 1 {
				s.WriteString("\n📊 Tüm Sorgular:\n\n")
				s.WriteString(m.table.View() + "\n")
			}
		}

		s.WriteString(helpStyle.Render("\n[Enter] Yeni Sorgu • [Q] Çıkış"))
	}

	return s.String()
}

// runTUI TUI modunu başlatır
func runTUI(apiKey string) error {
	p := tea.NewProgram(newTUIModel(apiKey), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
