package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
)

// TUI model
type tuiModel struct {
	state       tuiState
	textInput   textinput.Model
	spinner     spinner.Model
	table       table.Model
	results     []QueryResult
	currentMsg  string
	err         error
	width       int
	height      int
	apiKey      string
	queryDomain string
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

	return tuiModel{
		state:     stateInput,
		textInput: ti,
		spinner:   s,
		table:     t,
		results:   []QueryResult{},
		apiKey:    apiKey,
	}
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
		case "enter":
			if m.state == stateInput && m.textInput.Value() != "" {
				domain := strings.TrimSpace(m.textInput.Value())
				if isValidDomain(domain) {
					m.state = stateQuerying
					m.queryDomain = domain
					m.currentMsg = "Session başlatılıyor..."
					return m, tea.Batch(m.spinner.Tick, m.startQuery(domain))
				} else {
					m.err = fmt.Errorf("geçersiz domain: %s", domain)
				}
			} else if m.state == stateResult {
				// Yeni sorgu için input'a dön
				m.state = stateInput
				m.textInput.SetValue("")
				m.textInput.Focus()
				m.err = nil
				return m, textinput.Blink
			}
		case "esc":
			if m.state == stateResult {
				m.state = stateInput
				m.textInput.Focus()
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
		m.results = append(m.results, msg.result)
		m.updateTable()

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

	// Tablo güncellemesi
	if m.state == stateResult {
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
			s.WriteString("\n📊 Önceki Sorgular:\n\n")
			s.WriteString(m.table.View() + "\n")
		}

		s.WriteString(helpStyle.Render("\n[Enter] Sorgula • [Q] Çıkış"))

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
