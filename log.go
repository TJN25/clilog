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

var Log = NewLogger(0, "stdout")

func InitializeLogger(verbose int, logFile ...string) error {
	if Log != nil {
		if err := Log.Close(); err != nil {
			return err
		}
	}

	Log = NewLogger(verbose, logFile...)
	return nil
}

func CloseLogger() error {
	if Log == nil {
		return nil
	}
	return Log.Close()
}

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

func (l LogLevel) Name() string {
	if int(l) < 0 || int(l) >= len(logLevelNames) {
		return "UNKNOWN"
	}
	return logLevelNames[int(l)]
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
	output         io.Writer
	ShouldColorize bool
	CodeStyle      CodeStyle
	mu             sync.Mutex
}

type CodeStyle struct {
	Formatter string
	Style     string
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
		l.output = os.Stdout
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

	target := io.Writer(os.Stdout)

	if len(logFile) > 0 {
		if logFile[0] != "stdout" && logFile[0] != "" {
			f, err = os.OpenFile(logFile[0], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening log file: %s: %v. Defaulting to stdout.\n", logFile[0], err)
				f = nil
			}
			if f != nil {
				target = f
			}
		}
	}

	l := Logger{
		Level:      logLevel,
		outputFile: f,
		output:     target,
		CodeStyle: CodeStyle{
			Formatter: "terminal256",
			Style:     "catppuccin-latte",
		},
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
	l.writeSelected(msg)
}

func (l *Logger) Fprint(target io.Writer, msg string) {
	l.writeAlt(target, msg)
}

func (l *Logger) Fprintf(target io.Writer, msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	l.Fprint(target, msg)
}

func (l *Logger) Fprintln(target io.Writer, msg string) {
	l.Fprint(target, msg+"\n")
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
	var b strings.Builder

	for _, line := range lines.Lines {
		if !l.ShouldColorize || !isWriterTTY(l.output) {
			b.WriteString(line.Text)
		} else {
			b.WriteString(line.Style.Render(line.Text))
		}
	}

	l.Print(b.String())
}

func (l *Logger) Richln(lines StructuredTextBlock) {
	newLine := StyledText{Text: "\n", Style: MessageDefaultStyle}
	lines.Lines = append(lines.Lines, newLine)
	l.Rich(lines)
}

func (l *Logger) Frich(target io.Writer, lines StructuredTextBlock) {
	var b strings.Builder

	for _, line := range lines.Lines {
		if !l.ShouldColorize || !isWriterTTY(target) {
			b.WriteString(line.Text)
		} else {
			b.WriteString(line.Style.Render(line.Text))
		}
	}

	l.Fprint(target, b.String())
}

func (l *Logger) Code(msg, language, indent string) {
	lines := strings.Split(msg, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}

	msg = strings.Join(lines, "\n")

	if !l.ShouldColorize || !isWriterTTY(l.output) {
		l.Print(msg)
		return
	}

	var b strings.Builder
	err := quick.Highlight(&b, msg, language, l.CodeStyle.Formatter, l.CodeStyle.Style)
	if err != nil {
		l.Print(msg)
		return
	}

	l.Print(b.String())
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

func (l *Logger) writeSelected(msg string) {
	l.write(l.output, msg)
}

func (l *Logger) writeAlt(target io.Writer, msg string) {
	l.write(target, msg)
}

func (l *Logger) write(target io.Writer, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := fmt.Fprint(target, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger: error writing log message: %v\nOriginal message: %s", err, msg)
	}
}

// direct calls
// Package-level logging helpers using the global logger.

func Error(msg string) {
	Log.Error(msg)
}

func Errorf(msg string, args ...interface{}) {
	Log.Errorf(msg, args...)
}

func Errorln(msg string) {
	Log.Errorln(msg)
}

func Warn(msg string) {
	Log.Warn(msg)
}

func Warnf(msg string, args ...interface{}) {
	Log.Warnf(msg, args...)
}

func Warnln(msg string) {
	Log.Warnln(msg)
}

func Info(msg string) {
	Log.Info(msg)
}

func Infof(msg string, args ...interface{}) {
	Log.Infof(msg, args...)
}

func Infoln(msg string) {
	Log.Infoln(msg)
}

func Debug(msg string) {
	Log.Debug(msg)
}

func Debugf(msg string, args ...interface{}) {
	Log.Debugf(msg, args...)
}

func Debugln(msg string) {
	Log.Debugln(msg)
}

func Trace(msg string) {
	Log.Trace(msg)
}

func Tracef(msg string, args ...interface{}) {
	Log.Tracef(msg, args...)
}

func Traceln(msg string) {
	Log.Traceln(msg)
}

func Panic(msg string) {
	Log.Panic(msg)
}

func Panicf(msg string, args ...interface{}) {
	Log.Panicf(msg, args...)
}

func Panicln(msg string) {
	Log.Panicln(msg)
}

func Fatal(exitcode int, msg string) {
	Log.Fatal(exitcode, msg)
}

func Fatalf(exitcode int, msg string, args ...interface{}) {
	Log.Fatalf(exitcode, msg, args...)
}

func Fatalln(exitcode int, msg string) {
	Log.Fatalln(exitcode, msg)
}

func Print(msg string) {
	Log.Print(msg)
}

func Printf(msg string, args ...interface{}) {
	Log.Printf(msg, args...)
}

func Println(msg string) {
	Log.Println(msg)
}

func Fprint(target io.Writer, msg string) {
	Log.Fprint(target, msg)
}

func Fprintf(target io.Writer, msg string, args ...interface{}) {
	Log.Fprintf(target, msg, args...)
}

func Fprintln(target io.Writer, msg string) {
	Log.Fprintln(target, msg)
}

func Rich(lines StructuredTextBlock) {
	Log.Rich(lines)
}

func Richln(lines StructuredTextBlock) {
	Log.Richln(lines)
}

func Frich(target io.Writer, lines StructuredTextBlock) {
	Log.Frich(target, lines)
}

func Code(msg, language, indent string) {
	Log.Code(msg, language, indent)
}

func Codeln(msg, language, indent string) {
	Log.Codeln(msg, language, indent)
}

func GetErrorPipe() io.Writer {
	return Log.GetErrorPipe()
}
