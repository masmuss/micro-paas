// Package tui provides the terminal user interface for managing micro-paas instances.
package tui

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
)

type mode int

const (
	modeDashboard mode = iota
	modeLogs
)

var (
	// Color palette - modern terminal colors
	subtle  = lipgloss.AdaptiveColor{Light: "#7D7D7D", Dark: "#525252"}
	special = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
	warning = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}
	danger  = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}
	success = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}
	text    = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	muted   = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}

	// Base styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 0)

	panelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(special).
			MarginBottom(0)

	infoLabelStyle = lipgloss.NewStyle().
			Foreground(muted).
			Width(15)

	infoValueStyle = lipgloss.NewStyle().
			Foreground(text)

	statusRunningStyle = lipgloss.NewStyle().Foreground(success).Bold(true)
	statusStoppedStyle = lipgloss.NewStyle().Foreground(danger).Bold(true)
	statusPendingStyle = lipgloss.NewStyle().Foreground(warning).Bold(true)

	// Footer/Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1)
)

const (
	minWidth  = 100
	minHeight = 24
)

type modelTUI struct {
	repo      repository.InstanceRepository
	dockerSvc service.DockerService
	table     table.Model
	viewport  viewport.Model
	instances []*model.Instance
	selected  *model.Instance
	mode      mode
	err       error
	width     int
	height    int
	message   string

	logBuffer string
	logChan   chan string
	logCancel context.CancelFunc
}

// NewModel creates a new TUI model.
func NewModel(repo repository.InstanceRepository, dockerSvc service.DockerService) tea.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 6},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle).
		BorderBottom(true).
		Bold(true).
		Foreground(special)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	s.Cell = s.Cell.
		Padding(0, 1)
	t.SetStyles(s)

	v := viewport.New(0, 0)
	// We handle borders manually in View() for more control

	return modelTUI{
		repo:      repo,
		dockerSvc: dockerSvc,
		table:     t,
		viewport:  v,
		mode:      modeDashboard,
		logChan:   make(chan string),
	}
}

type tickMsg time.Time
type logLineMsg string

func (m modelTUI) Init() tea.Cmd {
	return tea.Batch(
		m.fetchInstances(),
		tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

func (m modelTUI) fetchInstances() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		instances, err := m.repo.List(ctx)
		if err != nil {
			return err
		}
		return instances
	}
}

func waitForLog(c chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-c
		if !ok {
			return nil
		}
		return logLineMsg(line)
	}
}

func (m *modelTUI) startStreaming(inst *model.Instance) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.logCancel = cancel

		reader, err := m.dockerSvc.GetContainerLogs(ctx, inst.ContainerID)
		if err != nil {
			return err
		}

		go func() {
			defer reader.Close()
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return
				case m.logChan <- scanner.Text():
				}
			}
		}()
		return nil
	}
}

func (m modelTUI) handleAction(action string, inst *model.Instance) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var err error
		switch action {
		case "start":
			err = m.dockerSvc.StartContainer(ctx, inst.ContainerID)
		case "stop":
			err = m.dockerSvc.StopContainer(ctx, inst.ContainerID)
		case "restart":
			_ = m.dockerSvc.StopContainer(ctx, inst.ContainerID)
			err = m.dockerSvc.StartContainer(ctx, inst.ContainerID)
		}

		if err != nil {
			return err
		}

		status, _ := m.dockerSvc.GetContainerStatus(ctx, inst.ContainerID)
		switch status {
		case "running":
			inst.Status = model.StatusRunning
		default:
			inst.Status = model.StatusStopped
		}
		_ = m.repo.Update(ctx, inst)

		return m.fetchInstances()()
	}
}

func (m modelTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.mode == modeLogs {
				m.stopLogs()
				return m, nil
			}
			return m, tea.Quit
		case "r":
			m.message = "Refreshing..."
			return m, m.fetchInstances()
		case "s":
			if m.mode == modeDashboard && m.selected != nil {
				m.message = "Starting " + m.selected.Name + "..."
				return m, m.handleAction("start", m.selected)
			}
		case "x":
			if m.mode == modeDashboard && m.selected != nil {
				m.message = "Stopping " + m.selected.Name + "..."
				return m, m.handleAction("stop", m.selected)
			}
		case "R":
			if m.mode == modeDashboard && m.selected != nil {
				m.message = "Restarting " + m.selected.Name + "..."
				return m, m.handleAction("restart", m.selected)
			}
		case "l":
			if m.mode == modeDashboard && m.selected != nil {
				m.mode = modeLogs
				m.logBuffer = " Connecting to Docker stream...\n"
				m.viewport.SetContent(m.logBuffer)
				return m, tea.Batch(m.startStreaming(m.selected), waitForLog(m.logChan))
			}
		case "esc":
			if m.mode == modeLogs {
				m.stopLogs()
				return m, nil
			}
		case "j":
			if m.mode == modeDashboard {
				m.table.MoveDown(1)
				m.updateSelected()
				return m, nil
			}
		case "k":
			if m.mode == modeDashboard {
				m.table.MoveUp(1)
				m.updateSelected()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(m.height - 14)
		// Precise sizing for viewport to avoid overlapping
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = m.height - 12

	case logLineMsg:
		if m.mode == modeLogs {
			line := string(msg)
			if len(line) > 8 && line[0] <= 2 {
				line = line[8:]
			}
			m.logBuffer += line + "\n"

			// Optional: limit buffer size to 1000 lines to prevent memory issues
			lines := strings.Split(m.logBuffer, "\n")
			if len(lines) > 1000 {
				m.logBuffer = strings.Join(lines[len(lines)-1000:], "\n")
			}

			m.viewport.SetContent(m.logBuffer)
			m.viewport.GotoBottom()
			return m, waitForLog(m.logChan)
		}

	case tickMsg:
		if m.mode == modeDashboard {
			cmds = append(cmds, m.fetchInstances())
		}
		cmds = append(cmds, tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}))

	case []*model.Instance:
		m.instances = msg
		m.message = ""
		rows := make([]table.Row, len(msg))
		for i, inst := range msg {
			statusStr := "[RUN]"
			statusStyle := statusRunningStyle
			if inst.Status != model.StatusRunning {
				statusStr = "[STP]"
				statusStyle = statusStoppedStyle
			}
			styledStatus := statusStyle.Render(statusStr)
			rows[i] = table.Row{
				fmt.Sprintf("%d", inst.ID),
				inst.Name,
				styledStatus,
			}
		}
		m.table.SetRows(rows)
		if len(msg) > 0 {
			m.updateSelected()
		}

	case error:
		m.err = msg
		m.message = ""
	}

	if m.mode == modeDashboard {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		m.updateSelected()
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *modelTUI) stopLogs() {
	if m.logCancel != nil {
		m.logCancel()
	}
	m.mode = modeDashboard
	m.logBuffer = ""
	m.viewport.SetContent("")
}

func (m *modelTUI) updateSelected() {
	if len(m.instances) == 0 {
		return
	}
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.instances) {
		m.selected = m.instances[idx]
	}
}

func (m modelTUI) View() string {
	if m.width < minWidth || m.height < minHeight {
		return fmt.Sprintf("\n  Terminal too small!\n  Current: %dx%d\n  Required: %dx%d",
			m.width, m.height, minWidth, minHeight)
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'r' to retry or 'q' to quit.", m.err)
	}

	if m.mode == modeLogs {
		return m.viewLogs()
	}

	return m.viewDashboard()
}

func (m modelTUI) viewDashboard() string {
	runningCount := 0
	for _, inst := range m.instances {
		if inst.Status == model.StatusRunning {
			runningCount++
		}
	}

	// Header full width
	header := headerStyle.Width(m.width - 2).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			"🚀 Micro PaaS Dashboard",
			lipgloss.NewStyle().Foreground(muted).Render(fmt.Sprintf(" | Apps: %d (%d running) | %s",
				len(m.instances), runningCount, time.Now().Format("15:04:05"))),
		),
	)

	// Left panel: table + system status
	leftPanelWidth := 45

	// Services panel with table
	servicesPanel := panelStyle.Width(leftPanelWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(fmt.Sprintf("📋 Services (%d)", len(m.instances))),
			m.table.View(),
		),
	)

	// System status panel
	systemPanel := panelStyle.Width(leftPanelWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("⚙ System Status"),
			renderInfoRow("System", statusRunningStyle.Render("● Online")),
			renderInfoRow("Network", statusRunningStyle.Render("● Connected")),
			renderInfoRow("Auth", statusRunningStyle.Render("● Active")),
		),
	)

	leftPanel := lipgloss.JoinVertical(lipgloss.Left,
		servicesPanel,
		"",
		systemPanel,
	)

	// Right panel: instance details
	rightPanelWidth := m.width - leftPanelWidth - 6
	var rightContent string

	basePanelStyle := panelStyle.Width(rightPanelWidth).Height(m.height - 10)

	if m.selected != nil {
		envLines := []string{}
		keys := make([]string, 0, len(m.selected.Env))
		for k := range m.selected.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rendered := lipgloss.NewStyle().Foreground(muted).Render("  "+k+"=") +
				infoValueStyle.Render(m.selected.Env[k])
			envLines = append(envLines, rendered)
		}

		statusText := statusRunningStyle.Render("● RUNNING")
		if m.selected.Status != model.StatusRunning {
			statusText = statusStoppedStyle.Render("● STOPPED")
		}

		rightContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("📦 Detail: "+m.selected.Name),
			"",
			renderInfoRow("ID", fmt.Sprintf("%d", m.selected.ID)),
			renderInfoRow("Name", m.selected.Name),
			renderInfoRow("Status", statusText),
			renderInfoRow("Container", m.selected.ContainerID[:12]),
			"",
			lipgloss.NewStyle().Foreground(special).Bold(true).Render("🌐 Network"),
			renderInfoRow("Proxy", fmt.Sprintf("%s.localhost → :%d", m.selected.Subdomain, m.selected.Port)),
			"",
			lipgloss.NewStyle().Foreground(special).Bold(true).Render("⚡ Resources"),
			renderInfoRow("CPU", renderProgressBar(0.42, 25)+" 42%"),
			renderInfoRow("MEM", renderProgressBar(0.18, 25)+" 180MB/1024MB"),
			"",
			lipgloss.NewStyle().Foreground(special).Bold(true).Render("🔧 Environment"),
			lipgloss.NewStyle().MarginLeft(2).Render(strings.Join(envLines, "\n")),
		)
	} else {
		rightContent = lipgloss.Place(
			rightPanelWidth-4,
			m.height-12,
			lipgloss.Center,
			lipgloss.Center,
			lipgloss.NewStyle().Foreground(muted).Render("← Select a service from the left panel"),
		)
	}

	rightPanel := basePanelStyle.Render(rightContent)

	// Main layout
	main := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Status message
	statusMsg := " "
	if m.message != "" {
		statusMsg = statusPendingStyle.Render("⚡ " + m.message)
	}

	// Footer
	footer := helpStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			"[q] quit",
			"• [r] refresh",
			"• [j/k] navigate",
			"• [s] start",
			"• [x] stop",
			"• [R] restart",
			"• [l] logs",
		),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, main, statusMsg, footer)
}

func (m modelTUI) viewLogs() string {
	header := headerStyle.Render(fmt.Sprintf(" 📝 Logs: %s | [esc/q] back to dashboard ", m.selected.Name))

	// Create a bordered box for logs manually to avoid recursive View() issues
	logBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(m.width - 2).
		Height(m.height - 6).
		Render(m.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		logBox,
		lipgloss.NewStyle().Faint(true).Padding(0, 1).Render(" Use arrows or pgup/pgdn to scroll log history "),
	)
}

func renderInfoRow(label, value string) string {
	return fmt.Sprintf("%s %s", infoLabelStyle.Render(label+":"), infoValueStyle.Render(value))
}

func renderProgressBar(percent float64, width int) string {
	fullSize := int(percent * float64(width))
	if fullSize > width {
		fullSize = width
	}
	if fullSize < 0 {
		fullSize = 0
	}

	full := strings.Repeat("█", fullSize)
	empty := strings.Repeat("░", width-fullSize)
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("57")).Render(full) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("237")).Render(empty)
	return bar
}
