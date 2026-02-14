package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleSystem = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	styleWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleError  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleTool   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleModel  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleUser   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	styleThread = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	styleName   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleToken  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleHelp   = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

func System(msg string) string { return styleSystem.Render(msg) }
func Warn(msg string) string   { return styleWarn.Render(msg) }
func Error(msg string) string  { return styleError.Render(msg) }
func Tool(msg string) string   { return styleTool.Render(msg) }
func Model(msg string) string  { return styleModel.Render(msg) }
func User(msg string) string   { return styleUser.Render(msg) }
func Thread(msg string) string { return styleThread.Render(msg) }
func Name(msg string) string   { return styleName.Render(msg) }
func Token(msg string) string  { return styleToken.Render(msg) }

func HelpCommand(cmd, desc string) string {
	left := styleHelp.Render(fmt.Sprintf("%-16s", cmd))
	return left + " - " + desc
}

func Prompt(tokens int64, model, thread string) string {
	thread = strings.TrimSpace(thread)
	if thread == "" {
		thread = "no-thread"
	}
	return fmt.Sprintf("%s %s %s > ", Token(fmt.Sprintf("tok:%d", tokens)), Name(model), Thread(thread))
}

func AssistantPrefix() string {
	return Model("assistant> ")
}

func ToolPrefix() string {
	return Tool("tool> ")
}
