package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	clog "github.com/charmbracelet/log"
)

var _ = os.Stderr // keep import

// color palette — dark theme friendly, tasteful
var (
	colorPrimary  = lipgloss.Color("#7C3AED") // purple
	colorDim      = lipgloss.Color("#6B7280") // gray
	colorGreen    = lipgloss.Color("#10B981")
	colorYellow   = lipgloss.Color("#F59E0B")
	colorBlue     = lipgloss.Color("#3B82F6")
	colorRed      = lipgloss.Color("#EF4444")
	colorCyan     = lipgloss.Color("#06B6D4")
	colorOrange   = lipgloss.Color("#F97316")
	colorWhite    = lipgloss.Color("#F9FAFB")
	colorMuted    = lipgloss.Color("#9CA3AF")
)

// styles
var (
	styleBanner = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 2).
			MarginTop(1).
			MarginBottom(0).
			MarginLeft(1)

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	styleVersion = lipgloss.NewStyle().
			Foreground(colorDim)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(8)

	styleValue = lipgloss.NewStyle().
			Foreground(colorWhite)

	styleURL = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleRouteSection = lipgloss.NewStyle().
				MarginLeft(1).
				MarginBottom(1)

	styleSeparator = lipgloss.NewStyle().
			Foreground(colorDim)

	styleMethod = map[string]lipgloss.Style{
		"GET":    lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Width(7),
		"POST":   lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Width(7),
		"PUT":    lipgloss.NewStyle().Foreground(colorBlue).Bold(true).Width(7),
		"PATCH":  lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Width(7),
		"DELETE": lipgloss.NewStyle().Foreground(colorRed).Bold(true).Width(7),
	}

	stylePath = lipgloss.NewStyle().
			Foreground(colorWhite)

	styleShutdown = lipgloss.NewStyle().
			Foreground(colorDim).
			MarginLeft(1).
			Italic(true)
)

// logger is the charmbracelet logger for request logging
var logger *clog.Logger

func init() {
	logger = clog.NewWithOptions(os.Stderr, clog.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
	})
}

// methodStyle returns the colored style for an HTTP method
func methodStyle(method string) lipgloss.Style {
	if s, ok := styleMethod[method]; ok {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Width(7)
}

// statusStyle returns a colored string for a status code
func statusStyle(code int) string {
	s := fmt.Sprintf("%d", code)
	switch {
	case code < 300:
		return lipgloss.NewStyle().Foreground(colorGreen).Render(s)
	case code < 400:
		return lipgloss.NewStyle().Foreground(colorCyan).Render(s)
	case code < 500:
		return lipgloss.NewStyle().Foreground(colorYellow).Render(s)
	default:
		return lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render(s)
	}
}

// timingStyle renders timing in dim gray
func timingStyle(d time.Duration) string {
	return lipgloss.NewStyle().Foreground(colorDim).Render(d.Round(time.Microsecond).String())
}

// renderBanner renders the startup banner for serve mode
func renderBanner(mode, specFile string, portNum int, seedVal int64, delayVal time.Duration, chaosMode, noAuthMode bool) string {
	var b strings.Builder

	// title line
	title := styleTitle.Render("⬛ portblock") + " " + styleVersion.Render("v"+version)
	if mode != "serve" {
		title = styleTitle.Render("⬛ portblock "+mode) + " " + styleVersion.Render("v"+version)
	}
	b.WriteString(title + "\n")

	// config
	b.WriteString(styleLabel.Render("spec") + styleValue.Render(specFile) + "\n")
	b.WriteString(styleLabel.Render("port") + styleValue.Render(fmt.Sprintf("%d", portNum)) + "\n")
	if mode == "serve" {
		b.WriteString(styleLabel.Render("seed") + styleValue.Render(fmt.Sprintf("%d", seedVal)) + "\n")
	}
	if delayVal > 0 {
		b.WriteString(styleLabel.Render("delay") + styleValue.Render(delayVal.String()) + "\n")
	}
	if chaosMode {
		b.WriteString(styleLabel.Render("chaos") + lipgloss.NewStyle().Foreground(colorOrange).Render("enabled 💥") + "\n")
	}
	if noAuthMode {
		b.WriteString(styleLabel.Render("auth") + lipgloss.NewStyle().Foreground(colorDim).Render("disabled") + "\n")
	}

	return styleBanner.Render(b.String())
}

// renderProxyBanner renders the startup banner for proxy mode
func renderProxyBanner(specFile, target string, portNum int, recordFile string) string {
	var b strings.Builder

	title := styleTitle.Render("⬛ portblock proxy") + " " + styleVersion.Render("v"+version)
	b.WriteString(title + "\n")

	b.WriteString(styleLabel.Render("spec") + styleValue.Render(specFile) + "\n")
	b.WriteString(styleLabel.Render("target") + styleURL.Render(target) + "\n")
	b.WriteString(styleLabel.Render("port") + styleValue.Render(fmt.Sprintf("%d", portNum)) + "\n")
	if recordFile != "" {
		b.WriteString(styleLabel.Render("record") + styleValue.Render(recordFile) + "\n")
	}

	return styleBanner.Render(b.String())
}

// renderReplayBanner renders the startup banner for replay mode
func renderReplayBanner(recordFile string, portNum, entries int) string {
	var b strings.Builder

	title := styleTitle.Render("⬛ portblock replay") + " " + styleVersion.Render("v"+version)
	b.WriteString(title + "\n")

	b.WriteString(styleLabel.Render("file") + styleValue.Render(recordFile) + "\n")
	b.WriteString(styleLabel.Render("port") + styleValue.Render(fmt.Sprintf("%d", portNum)) + "\n")
	b.WriteString(styleLabel.Render("entries") + styleValue.Render(fmt.Sprintf("%d", entries)) + "\n")

	return styleBanner.Render(b.String())
}

// renderRoutes renders the route list
func renderRoutes(routes []routeInfo) string {
	var b strings.Builder

	sep := styleSeparator.Render(strings.Repeat("─", 44))
	b.WriteString("  " + sep + "\n")

	for _, r := range routes {
		methods := ""
		for i, m := range r.methods {
			if i > 0 {
				methods += styleSeparator.Render(",")
			}
			methods += methodStyle(m).Render(m)
		}
		b.WriteString("  " + methods + stylePath.Render(r.path) + "\n")
	}
	b.WriteString("  " + sep + "\n")

	return b.String()
}

// renderReady renders the ready message
func renderReady(portNum int) string {
	url := fmt.Sprintf("http://localhost:%d", portNum)
	return lipgloss.NewStyle().MarginLeft(1).MarginBottom(1).Render(
		lipgloss.NewStyle().Foreground(colorGreen).Render("● ") +
			lipgloss.NewStyle().Foreground(colorMuted).Render("ready at ") +
			styleURL.Render(url),
	) + "\n"
}

// renderShutdown renders the shutdown message
func renderShutdown() string {
	return "\n" + styleShutdown.Render("● shutting down...") + "\n"
}

// routeInfo holds route data for rendering
type routeInfo struct {
	path    string
	methods []string
}

// logRequest logs a request with beautiful formatting
func logRequest(method, path string, status int, dur time.Duration) {
	m := methodStyle(method).Render(method)
	s := statusStyle(status)
	t := timingStyle(dur)
	p := stylePath.Render(path)

	fmt.Printf("  %s %s %s %s\n", m, p, s, t)
}

// logRequestValidationError logs a validation failure
func logRequestValidationError(method, path string, dur time.Duration) {
	m := methodStyle(method).Render(method)
	s := statusStyle(400)
	t := timingStyle(dur)
	p := stylePath.Render(path)
	msg := lipgloss.NewStyle().Foreground(colorYellow).Render("validation failed")

	fmt.Printf("  %s %s %s %s %s\n", m, p, s, msg, t)
}

// logChaos logs a chaos mode hit
func logChaos(method, path string, dur time.Duration) {
	m := methodStyle(method).Render(method)
	s := statusStyle(500)
	t := timingStyle(dur)
	p := stylePath.Render(path)
	chaos := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("CHAOS 💥")

	fmt.Printf("  %s %s %s %s %s\n", m, p, s, chaos, t)
}

// logProxyValidation logs proxy validation warnings
func logProxyValidation(kind, method, path string, err error) {
	m := methodStyle(method).Render(method)
	p := stylePath.Render(path)
	warn := lipgloss.NewStyle().Foreground(colorYellow).Render("⚠ " + kind)
	detail := lipgloss.NewStyle().Foreground(colorDim).Render(err.Error())

	fmt.Printf("  %s %s %s\n    %s\n", m, p, warn, detail)
}
