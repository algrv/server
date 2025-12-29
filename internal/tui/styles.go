package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	colorWhite        = lipgloss.Color("#FFFFFF")
	colorLightGray    = lipgloss.Color("#CCCCCC")
	colorGray         = lipgloss.Color("#888888")
	colorDarkGray     = lipgloss.Color("#444444")
	colorVeryDarkGray = lipgloss.Color("#222222")
	colorPurple       = lipgloss.Color("#8524a6")
	colorGreen        = lipgloss.Color("#00FF00")
	colorYellow       = lipgloss.Color("#FFFF00")
	colorRed          = lipgloss.Color("#FF0000")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Align(lipgloss.Center).
			MarginTop(1).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorLightGray).
			Align(lipgloss.Center).
			MarginBottom(2)

	commandStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	commandDescStyle = lipgloss.NewStyle().
				Foreground(colorGray).
				PaddingLeft(1)

	inputStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(colorLightGray)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDarkGray).
			Italic(true).
			MarginTop(1)
)

const logo = `
   █████╗ ██╗      ██████╗  ██████╗ ██████╗  █████╗ ██╗   ██╗███████╗
  ██╔══██╗██║     ██╔════╝ ██╔═══██╗██╔══██╗██╔══██╗██║   ██║██╔════╝
  ███████║██║     ██║  ███╗██║   ██║██████╔╝███████║██║   ██║█████╗
  ██╔══██║██║     ██║   ██║██║   ██║██╔══██╗██╔══██║╚██╗ ██╔╝██╔══╝
  ██║  ██║███████╗╚██████╔╝╚██████╔╝██║  ██║██║  ██║ ╚████╔╝ ███████╗
  ╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝
`
