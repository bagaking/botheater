package utils

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

func SPrintWithFrameCard(title, content string, maxWidth int) string {
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
	border := strings.Repeat("═", maxLength+lineNumberWidth+5)
	frame := fmt.Sprintf("╔%s╗\n║ %s%s ║\n╠%s╣\n",
		border,
		title,
		strings.Repeat(" ", maxLength+lineNumberWidth+3-runewidth.StringWidth(title)), // 3 = ' | '
		border,
	)
	for i, line := range lines {
		lineNumber := fmt.Sprintf("%*d", lineNumberWidth, i+1)
		line = wrapText(line, maxLength)
		for _, wrappedLine := range strings.Split(line, "\n") {
			frame += fmt.Sprintf("║ %s | %s%s ║\n", lineNumber, wrappedLine, strings.Repeat(" ", maxLength-runewidth.StringWidth(wrappedLine)))
			lineNumber = strings.Repeat(" ", lineNumberWidth) // 之后的行号为空格填充
		}
	}
	frame += fmt.Sprintf("╚%s╝", border)
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
