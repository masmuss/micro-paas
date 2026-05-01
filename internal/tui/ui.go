// Package tui provides the terminal user interface for managing micro-paas instances.
package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

const (
	minWidth  = 80
	minHeight = 24
)

type modelTUI struct {
	repo      repository.InstanceRepository
	dockerSvc service.DockerService
	table     table.Model
	instances []*model.Instance
	err       error
	width     int
	height    int
}

// NewModel creates a new TUI model with the given repository and docker service.
func NewModel(repo repository.InstanceRepository, dockerSvc service.DockerService) tea.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Name", Width: 20},
		{Title: "Subdomain", Width: 15},
		{Title: "Port", Width: 6},
		{Title: "Status", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return modelTUI{
		repo:      repo,
		dockerSvc: dockerSvc,
		table:     t,
	}
}

// Init initializes the TUI model.
func (m modelTUI) Init() tea.Cmd {
	return m.fetchInstances()
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

// Update handles UI updates based on messages.
func (m modelTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r": // Manual refresh
			return m, m.fetchInstances()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 4)
	case []*model.Instance:
		m.instances = msg
		rows := make([]table.Row, len(msg))
		for i, inst := range msg {
			rows[i] = table.Row{
				fmt.Sprintf("%d", inst.ID),
				inst.Name,
				inst.Subdomain,
				fmt.Sprintf("%d", inst.Port),
				inst.Status.String(),
			}
		}
		m.table.SetRows(rows)
	case error:
		m.err = msg
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the TUI view.
func (m modelTUI) View() string {
	if m.width < minWidth || m.height < minHeight {
		return fmt.Sprintf("\n  Terminal too small!\n  Current: %dx%d\n  Required: %dx%d",
			m.width, m.height, minWidth, minHeight)
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'q' to quit.", m.err)
	}

	s := lipgloss.NewStyle().Padding(1).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("57")).Render(" 🚀 Micro PaaS Dashboard "),
			baseStyle.Render(m.table.View()),
			lipgloss.NewStyle().Faint(true).Render(" [q] quit • [r] refresh • [j/k] navigate "),
		),
	)
	return s
}
