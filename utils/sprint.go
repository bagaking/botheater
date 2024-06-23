package utils

import (
	"fmt"
	"strings"
	"unicode"

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

	LiteLevel uint8
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

	StyFunctionStack = FrameStyle{
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
		Horizontal:  "─",
		Vertical:    "│",
		TitlePrefix: "🔍 ",
		LinePrefix:  "",

		LiteLevel: 1,
	}

	StyMsgCard = FrameStyle{
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
		Horizontal:  "─",
		Vertical:    "│",
		TitlePrefix: "",
		LinePrefix:  "",

		LiteLevel: 2,
	}
)

// SPrintWithCallStack 打印函数调用栈
func SPrintWithCallStack(title, content string, maxWidth int) string {
	return SPrintWithFrameCard(title, content, maxWidth, StyFunctionStack)
}

func SPrintWithMsgCard(title, content string, maxWidth int) string {
	return SPrintWithFrameCard(title, content, maxWidth, StyMsgCard)
}

// SPrintWithFrameCard 打印带框架的卡片
func SPrintWithFrameCard(title, content string, maxWidth int, style FrameStyle) string {
	lines := strings.Split(content, "\n")

	maxLength := runewidth.StringWidth(style.TitlePrefix + title)
	for i := range lines {
		lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
		lineLength := runewidth.StringWidth(lines[i])
		if lineLength > maxLength {
			maxLength = lineLength
		}
	}
	if maxWidth > 0 && maxLength > maxWidth {
		maxLength = maxWidth
	}

	lineNumberWidth := len(fmt.Sprintf("%d", len(lines)))
	border := strings.Repeat(style.Horizontal, maxLength+lineNumberWidth+5)

	frame := ""
	switch style.LiteLevel {
	case 0:
		titleHead := fmt.Sprintf("%s%s%s\n", style.TopLeft, border, style.TopRight)
		titleLine := fmt.Sprintf("%s %s%s %s%s\n",
			style.Vertical,
			style.TitlePrefix, title,
			strings.Repeat(" ", maxLength+lineNumberWidth+3-runewidth.StringWidth(style.TitlePrefix+title)), style.Vertical, // 3 = ' | '
		)
		titleGround := fmt.Sprintf("%s%s%s\n", style.Vertical, border, style.Vertical)
		frame = titleHead + titleLine + titleGround
	default:
		repeat := maxLength + lineNumberWidth + 2 - runewidth.StringWidth(style.TitlePrefix+title)
		if repeat < 0 {
			repeat = 0
		}
		titleLine := fmt.Sprintf("%s%s %s%s %s%s\n",
			style.TopLeft, style.Horizontal,
			style.TitlePrefix, title,
			strings.Repeat(style.Horizontal, repeat), style.TopRight, // 3 = ' | '
		)
		frame = titleLine
	}

	for i, line := range lines {
		lineNumber := fmt.Sprintf("%*d", lineNumberWidth, i+1)
		line = wrapText(line, maxLength)
		wrapped := strings.Split(line, "\n")
		for j, wrappedLine := range wrapped {
			bl, rl, fill := style.Vertical, style.Vertical, " "
			if style.LiteLevel >= 2 && i == len(lines)-1 && j == len(wrapped)-1 {
				bl, rl, fill = style.BottomLeft, style.BottomRight, style.Horizontal
			}
			frame += fmt.Sprintf("%s %s | %s %s%s\n", bl,
				lineNumber, wrappedLine,
				strings.Repeat(fill, maxLength-runewidth.StringWidth(wrappedLine)), rl)
			lineNumber = strings.Repeat(" ", lineNumberWidth) // 之后的行号为空格填充
		}
	}
	if style.LiteLevel < 2 {
		frame += fmt.Sprintf("%s%s%s\n", style.BottomLeft, border, style.BottomRight)
	}

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
