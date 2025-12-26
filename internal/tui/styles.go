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
	colorBlack        = lipgloss.Color("#000001")
	colorDarkGreen    = lipgloss.Color("#020f0e")
	colorPurple       = lipgloss.Color("#8524a6")
	colorGreen        = lipgloss.Color("#00FF00")
	colorYellow       = lipgloss.Color("#FFFF00")
	colorRed          = lipgloss.Color("#FF0000")
	colorTransparent  = lipgloss.Color("tansparent")
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

	menuItemStyle = lipgloss.NewStyle().
			Foreground(colorLightGray).
			PaddingLeft(2)

	menuItemSelectedStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Bold(true).
				PaddingLeft(2)

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

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorGray).
			Padding(0)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorLightGray).
			Bold(true)

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
