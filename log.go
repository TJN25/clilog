package clilog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
)

var (
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Bold(true)

	RedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	YellowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	GreenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	BlueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	BlueBoldStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	BlueItalicStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Italic(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")) // Yellow

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // Green

	DebugStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")) // Blue

	TraceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Red
			Bold(true)

	TimestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#707070", Dark: "#707070"})

	MessageDefaultStyle = lipgloss.NewStyle()
)

type LogLevel int

const (
	LogAlways LogLevel = iota
	LogError
	LogWarn
	LogInfo
	LogDebug
	LogTrace
)

var logLevelNames = [...]string{
	"ALWAYS", // 0
	"ERROR",  // 1
	"WARN",   // 2
	"INFO",   // 3
	"DEBUG",  // 4
	"TRACE",  // 5
}

func (l *LogLevel) Name() string {
	name := logLevelNames[int(*l)]
	return name
}

type StyledText struct {
	Text  string
	Style lipgloss.Style
}

type StructuredTextBlock struct {
	Lines []StyledText
}

type LoggerPanic struct {
	Message string
}

type Logger struct {
	Level          LogLevel
	outputFile     *os.File
	ShouldColorize bool
	mu             sync.Mutex
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var err error
	if l.outputFile != nil {
		err = l.outputFile.Close()
		l.outputFile = nil
	}
	return err
}

func NewLogger(verbosity int, logFile ...string) *Logger {
	if verbosity < 0 {
		verbosity = 0
	} else if verbosity > int(LogTrace) {
		verbosity = int(LogTrace)
	}
	logLevel := LogLevel(verbosity)

	var f *os.File
	var err error
	var colorize bool = true

	if len(logFile) > 0 {
		if logFile[0] != "stdout" && logFile[0] != "" {
			f, err = os.OpenFile(logFile[0], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening log file: %s: %v. Defaulting to stdout.\n", logFile[0], err)
				f = nil
			}
		}
	}

	l := Logger{
		Level:          logLevel,
		outputFile:     f,
		ShouldColorize: colorize,
	}
	return &l
}

func (l *Logger) Error(msg string) {
	if l.Level >= LogError {
		prefix := StyledText{
			Text:  "ERROR: ",
			Style: ErrorStyle,
		}
		timeStamp := timeSegment()
		styledMsg := StyledText{
			Text:  msg,
			Style: MessageDefaultStyle,
		}
		lines := StructuredTextBlock{
			Lines: []StyledText{prefix, timeStamp, styledMsg},
		}
		if l.outputFile != nil {
			l.Frich(l.outputFile, lines)
		}
		l.Frich(os.Stderr, lines)
	}
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	if l.Level >= LogError {
		msg = fmt.Sprintf(msg, args...)
		l.Error(msg)
	}
}

func (l *Logger) Errorln(msg string) {
	if l.Level >= LogError {
		msg += "\n"
		l.Error(msg)
	}
}

func (l *Logger) Warn(msg string) {
	if l.Level >= LogWarn {
		prefix := StyledText{
			Text:  "Warn: ",
			Style: WarningStyle,
		}
		timeStamp := timeSegment()
		styledMsg := StyledText{
			Text:  msg,
			Style: MessageDefaultStyle,
		}
		lines := StructuredTextBlock{
			Lines: []StyledText{prefix, timeStamp, styledMsg},
		}
		if l.outputFile != nil {
			l.Frich(l.outputFile, lines)
		}
		l.Frich(os.Stderr, lines)
	}
}

func (l *Logger) Warnf(msg string, args ...interface{}) {
	if l.Level >= LogWarn {
		msg = fmt.Sprintf(msg, args...)
		l.Warn(msg)
	}
}

func (l *Logger) Warnln(msg string) {
	if l.Level >= LogWarn {
		msg += "\n"
		l.Warn(msg)
	}
}

func (l *Logger) Info(msg string) {
	if l.Level >= LogInfo {
		prefix := StyledText{
			Text:  "INFO: ",
			Style: InfoStyle,
		}
		timeStamp := timeSegment()
		styledMsg := StyledText{
			Text:  msg,
			Style: MessageDefaultStyle,
		}
		lines := StructuredTextBlock{
			Lines: []StyledText{prefix, timeStamp, styledMsg},
		}
		l.Rich(lines)
	}
}

func (l *Logger) Infof(msg string, args ...interface{}) {
	if l.Level >= LogInfo {
		msg = fmt.Sprintf(msg, args...)
		l.Info(msg)
	}
}

func (l *Logger) Infoln(msg string) {
	if l.Level >= LogInfo {
		msg += "\n"
		l.Info(msg)
	}
}

func (l *Logger) Debug(msg string) {
	if l.Level >= LogDebug {
		prefix := StyledText{
			Text:  "DEBUG: ",
			Style: DebugStyle,
		}
		timeStamp := timeSegment()
		styledMsg := StyledText{
			Text:  msg,
			Style: MessageDefaultStyle,
		}
		lines := StructuredTextBlock{
			Lines: []StyledText{prefix, timeStamp, styledMsg},
		}
		l.Rich(lines)
	}
}

func (l *Logger) Debugf(msg string, args ...interface{}) {
	if l.Level >= LogDebug {
		msg = fmt.Sprintf(msg, args...)
		l.Debug(msg)
	}
}

func (l *Logger) Debugln(msg string) {
	if l.Level >= LogDebug {
		msg += "\n"
		l.Debug(msg)
	}
}

func (l *Logger) Trace(msg string) {
	if l.Level >= LogTrace {
		prefix := StyledText{
			Text:  "TRACE: ",
			Style: MessageDefaultStyle,
		}
		timeStamp := timeSegment()
		styledMsg := StyledText{
			Text:  msg,
			Style: MessageDefaultStyle,
		}
		lines := StructuredTextBlock{
			Lines: []StyledText{prefix, timeStamp, styledMsg},
		}
		l.Rich(lines)
	}
}

func (l *Logger) Tracef(msg string, args ...interface{}) {
	if l.Level >= LogTrace {
		msg = fmt.Sprintf(msg, args...)
		l.Trace(msg)
	}
}

func (l *Logger) Traceln(msg string) {
	if l.Level >= LogTrace {
		msg += "\n"
		l.Trace(msg)
	}
}

func (l *Logger) Panic(msg string) {
	styledMsg := StyledText{
		Text:  msg,
		Style: MessageDefaultStyle,
	}
	var lines StructuredTextBlock
	if l.Level >= LogDebug {
		prefix := StyledText{
			Text:  "PANIC: ",
			Style: ErrorStyle,
		}
		timeStamp := timeSegment()
		lines = StructuredTextBlock{
			Lines: []StyledText{prefix, timeStamp, styledMsg},
		}
	} else {
		prefix := StyledText{
			Text:  "Error: ",
			Style: ErrorStyle,
		}
		lines = StructuredTextBlock{
			Lines: []StyledText{prefix, styledMsg},
		}
	}
	if l.outputFile != nil {
		l.Frich(l.outputFile, lines)
	}
	l.Frich(os.Stderr, lines)

	panic(LoggerPanic{Message: msg})
}

func (l *Logger) Panicf(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	l.Panic(msg)
}

func (l *Logger) Panicln(msg string) {
	msg += "\n"
	l.Panic(msg)
}

func (l *Logger) Fatal(exitcode int, msg string) {
	styledMsg := StyledText{
		Text:  msg,
		Style: MessageDefaultStyle,
	}
	var lines StructuredTextBlock
	prefix := StyledText{
		Text:  "FATAL: ",
		Style: ErrorStyle,
	}
	timeStamp := timeSegment()
	lines = StructuredTextBlock{
		Lines: []StyledText{prefix, timeStamp, styledMsg},
	}
	if l.outputFile != nil {
		l.Frich(l.outputFile, lines)
	}
	l.Frich(os.Stderr, lines)
	os.Exit(exitcode)
}

func (l *Logger) Fatalf(exitcode int, msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	l.Fatal(exitcode, msg)
}

func (l *Logger) Fatalln(exitcode int, msg string) {
	msg += "\n"
	l.Fatal(exitcode, msg)
}

func (l *Logger) Print(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.outputFile != nil {
		_, err := fmt.Fprint(l.outputFile, msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Logger: Error writing to log file: %v\nOriginal message: %s", err, msg)
		}
	} else {
		fmt.Print(msg)
	}
}

func (l *Logger) printNoLock(msg string) {
	if l.outputFile != nil {
		_, err := fmt.Fprint(l.outputFile, msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Logger: Error writing to log file: %v\nOriginal message: %s", err, msg)
		}
	} else {
		fmt.Print(msg)
	}
}

func (l *Logger) Fprint(target io.Writer, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := fmt.Fprint(target, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger: Error writing to %s: file: %v\nOriginal message: %s", target, err, msg)
	}
}

func (l *Logger) Fprintf(target io.Writer, msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	l.Fprint(target, msg)
}

func (l *Logger) Fprintln(target io.Writer, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg += "\n"
	_, err := fmt.Fprint(target, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger: Error writing to %s: file: %v\nOriginal message: %s", target, err, msg)
	}
}

func (l *Logger) fPrintNoLock(target io.Writer, msg string) {
	_, err := fmt.Fprint(target, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger: Error writing to %s: file: %v\nOriginal message: %s", target, err, msg)
	}
}

func (l *Logger) Printf(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	l.Print(msg)
}

func (l *Logger) Println(msg string) {
	msg += "\n"
	l.Print(msg)
}

func (l *Logger) Rich(lines StructuredTextBlock) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range lines.Lines {
		if !l.ShouldColorize || l.outputFile != nil {
			l.printNoLock(line.Text)
		} else {
			l.printNoLock(line.Style.Render(line.Text))
		}
	}
}

func (l *Logger) Richln(lines StructuredTextBlock) {
	newLine := StyledText{Text: "\n", Style: MessageDefaultStyle}
	lines.Lines = append(lines.Lines, newLine)
	l.Rich(lines)
}

func (l *Logger) Frich(target io.Writer, lines StructuredTextBlock) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range lines.Lines {
		var msg string
		if !l.ShouldColorize || !isWriterTTY(target) {
			msg = line.Text
		} else {
			msg = line.Style.Render(line.Text)
		}
		l.fPrintNoLock(target, msg)
	}
}

func (l *Logger) Code(msg, language, indent string) {
	formatter := "terminal256"
	style := "catppuccin-latte"

	if !l.ShouldColorize {
		l.Print(msg)
		return
	}
	lines := strings.Split(msg, "\n")
	newLines := make([]string, 0)
	for _, line := range lines {
		if line != "" {
			line = indent + line
		}
		newLines = append(newLines, line)
	}
	msg = strings.Join(newLines, "\n")

	_ = quick.Highlight(os.Stdout, msg, language, formatter, style)
}

func (l *Logger) Codeln(msg, language, indent string) {
	msg += "\n"
	l.Code(msg, language, indent)
}

func (l *Logger) GetErrorPipe() io.Writer {
	if l.outputFile != nil {
		return l.outputFile
	}
	return os.Stderr
}

func timeSegment() StyledText {
	msg := StyledText{
		Text:  time.Now().Format("15:04:05") + " ",
		Style: TimestampStyle,
	}
	return msg

}

func isWriterTTY(writer io.Writer) bool {
	if f, ok := writer.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false // Could not get stat, assume not a TTY
		}
		return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	}
	return false
}
