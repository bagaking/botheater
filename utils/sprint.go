package utils

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// FrameStyle 定义框架样式
type FrameStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	TitlePrefix string
	LinePrefix  string
}

// 预定义的样式
var (
	SimpleStyle = FrameStyle{
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
		Horizontal:  "═",
		Vertical:    "║",
		TitlePrefix: "",
		LinePrefix:  "",
	}

	StyTalk = FrameStyle{
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
		Horizontal:  "═",
		Vertical:    "║",
		TitlePrefix: "🚗 ",
		LinePrefix:  "",
	}

	CallStackStyle = FrameStyle{
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
		Horizontal:  "═",
		Vertical:    "║",
		TitlePrefix: "🔍 ",
		LinePrefix:  "",
	}
)

// SPrintWithCallStack 打印函数调用栈
func SPrintWithCallStack(title, content string, maxWidth int) string {
	return SPrintWithFrameCard(title, content, maxWidth, CallStackStyle)
}

// SPrintWithFrameCard 打印带框架的卡片
func SPrintWithFrameCard(title, content string, maxWidth int, style FrameStyle) string {
	lines := strings.Split(content, "\n")

	maxLength := runewidth.StringWidth(title)
	for _, line := range lines {
		lineLength := runewidth.StringWidth(line)
		if lineLength > maxLength {
			maxLength = lineLength
		}
	}
	if maxWidth > 0 && maxLength > maxWidth {
		maxLength = maxWidth
	}

	lineNumberWidth := len(fmt.Sprintf("%d", len(lines)))
	border := strings.Repeat(style.Horizontal, maxLength+lineNumberWidth+5)
	frame := fmt.Sprintf("%s%s%s\n%s %s%s %s\n%s%s%s\n",
		style.TopLeft,
		border,
		style.TopRight,
		style.Vertical,
		style.TitlePrefix,
		title,
		strings.Repeat(" ", maxLength+lineNumberWidth+3-runewidth.StringWidth(title)), // 3 = ' | '
		style.Vertical,
		border,
		style.Vertical,
	)
	for i, line := range lines {
		lineNumber := fmt.Sprintf("%*d", lineNumberWidth, i+1)
		line = wrapText(line, maxLength)
		for _, wrappedLine := range strings.Split(line, "\n") {
			frame += fmt.Sprintf("%s %s | %s%s %s\n", style.Vertical, lineNumber, wrappedLine, strings.Repeat(" ", maxLength-runewidth.StringWidth(wrappedLine)), style.Vertical)
			lineNumber = strings.Repeat(" ", lineNumberWidth) // 之后的行号为空格填充
		}
	}
	frame += fmt.Sprintf("%s%s%s\n", style.BottomLeft, border, style.BottomRight)
	return frame
}

func wrapText(text string, maxWidth int) string {
	if runewidth.StringWidth(text) <= maxWidth {
		return text
	}
	var wrappedText strings.Builder
	var currentLine strings.Builder
	currentWidth := 0
	for _, r := range text {
		rw := runewidth.RuneWidth(r)
		if currentWidth+rw > maxWidth {
			wrappedText.WriteString(currentLine.String() + "\n")
			currentLine.Reset()
			currentWidth = 0
		}
		currentLine.WriteRune(r)
		currentWidth += rw
	}
	wrappedText.WriteString(currentLine.String())
	return wrappedText.String()
}
