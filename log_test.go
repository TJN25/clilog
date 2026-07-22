package clilog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var timestampPattern = regexp.MustCompile(`\d{2}:\d{2}:\d{2}`)

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func newBufferLogger(level LogLevel) (*Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := &Logger{
		Level:          level,
		output:         &buf,
		ShouldColorize: false,
		CodeStyle: CodeStyle{
			Formatter: "terminal256",
			Style:     "catppuccin-latte",
		},
	}
	return logger, &buf
}

func newFileLogger(t *testing.T, level LogLevel) (*Logger, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "clilog-test.log")
	logger := NewLogger(int(level), path)
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	logger.ShouldColorize = false
	t.Cleanup(func() {
		if err := logger.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Fatalf("closing logger: %v", err)
		}
	})
	return logger, path
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(content)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stderr pipe: %v", err)
	}
	os.Stderr = writer

	defer func() {
		os.Stderr = original
		_ = writer.Close()
		_ = reader.Close()
	}()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("closing stderr writer: %v", err)
	}
	os.Stderr = original

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading stderr: %v", err)
	}
	return string(content)
}

func mustPanicLoggerPanic(t *testing.T, fn func()) LoggerPanic {
	t.Helper()

	var got LoggerPanic
	didPanic := false
	func() {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			didPanic = true
			panicValue, ok := r.(LoggerPanic)
			if !ok {
				t.Fatalf("panic type = %T, want LoggerPanic", r)
			}
			got = panicValue
		}()
		fn()
	}()

	if !didPanic {
		t.Fatal("function did not panic")
	}
	return got
}

func withGlobalLogger(t *testing.T, logger *Logger) {
	t.Helper()

	original := Log
	Log = logger
	t.Cleanup(func() {
		if Log != nil && Log != logger {
			_ = Log.Close()
		}
		Log = original
	})
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}

func assertNotContains(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("expected %q not to contain %q", got, want)
	}
}

func assertTimestamp(t *testing.T, got string) {
	t.Helper()
	if !timestampPattern.MatchString(got) {
		t.Fatalf("expected %q to contain a timestamp shaped like HH:MM:SS", got)
	}
}

func TestLogLevelName(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{LogAlways, "ALWAYS"},
		{LogError, "ERROR"},
		{LogWarn, "WARN"},
		{LogInfo, "INFO"},
		{LogDebug, "DEBUG"},
		{LogTrace, "TRACE"},
		{LogLevel(-1), "UNKNOWN"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.Name(); got != tt.want {
				t.Fatalf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	t.Run("verbosity clamps", func(t *testing.T) {
		if got := NewLogger(-1, "stdout").Level; got != LogAlways {
			t.Fatalf("negative verbosity level = %v, want %v", got, LogAlways)
		}
		if got := NewLogger(999, "stdout").Level; got != LogTrace {
			t.Fatalf("high verbosity level = %v, want %v", got, LogTrace)
		}
	})

	t.Run("stdout targets do not open files", func(t *testing.T) {
		for _, target := range []string{"", "stdout"} {
			logger := NewLogger(int(LogInfo), target)
			if logger == nil {
				t.Fatal("NewLogger returned nil")
			}
			if logger.outputFile != nil {
				t.Fatalf("outputFile = %v, want nil", logger.outputFile)
			}
			if logger.output != os.Stdout {
				t.Fatalf("output = %v, want os.Stdout", logger.output)
			}
		}
	})

	t.Run("file target creates and appends", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "app.log")
		if err := os.WriteFile(path, []byte("existing\n"), 0664); err != nil {
			t.Fatalf("seeding file: %v", err)
		}

		logger := NewLogger(int(LogInfo), path)
		if logger.outputFile == nil {
			t.Fatal("outputFile = nil, want open file")
		}
		t.Cleanup(func() { _ = logger.Close() })

		logger.Print("new\n")
		if got := readFile(t, path); got != "existing\nnew\n" {
			t.Fatalf("file content = %q", got)
		}
	})

	t.Run("invalid file target falls back to stdout", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing", "app.log")
		logger := NewLogger(int(LogInfo), path)
		if logger == nil {
			t.Fatal("NewLogger returned nil")
		}
		if logger.outputFile != nil {
			t.Fatalf("outputFile = %v, want nil", logger.outputFile)
		}
		if logger.output != os.Stdout {
			t.Fatalf("output = %v, want os.Stdout", logger.output)
		}
	})
}

func TestCloseInitializeAndCloseLogger(t *testing.T) {
	t.Run("nil close", func(t *testing.T) {
		var logger *Logger
		if err := logger.Close(); err != nil {
			t.Fatalf("nil Close() error = %v", err)
		}
	})

	t.Run("file logger close resets output", func(t *testing.T) {
		logger, path := newFileLogger(t, LogInfo)
		file := logger.outputFile

		if err := logger.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if logger.outputFile != nil {
			t.Fatalf("outputFile = %v, want nil", logger.outputFile)
		}
		if logger.output != os.Stdout {
			t.Fatalf("output = %v, want os.Stdout", logger.output)
		}
		if _, err := file.WriteString("after close"); err == nil {
			t.Fatal("write to closed file succeeded")
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected log file to remain on disk: %v", err)
		}
	})

	t.Run("CloseLogger with nil global", func(t *testing.T) {
		original := Log
		Log = nil
		t.Cleanup(func() { Log = original })

		if err := CloseLogger(); err != nil {
			t.Fatalf("CloseLogger() error = %v", err)
		}
	})

	t.Run("InitializeLogger replaces global logger", func(t *testing.T) {
		original := Log
		oldLogger, oldPath := newFileLogger(t, LogInfo)
		Log = oldLogger
		t.Cleanup(func() {
			_ = CloseLogger()
			Log = original
		})

		newPath := filepath.Join(t.TempDir(), "new.log")
		if err := InitializeLogger(int(LogDebug), newPath); err != nil {
			t.Fatalf("InitializeLogger() error = %v", err)
		}
		if Log == oldLogger {
			t.Fatal("global Log still points to old logger")
		}
		if Log.Level != LogDebug {
			t.Fatalf("global level = %v, want %v", Log.Level, LogDebug)
		}
		if Log.outputFile == nil {
			t.Fatal("new global logger has nil outputFile")
		}
		if oldLogger.outputFile != nil {
			t.Fatal("old logger outputFile was not cleared")
		}
		if _, err := os.Stat(oldPath); err != nil {
			t.Fatalf("expected old log file to remain on disk: %v", err)
		}
	})
}

func TestPlainPrintMethods(t *testing.T) {
	logger, selected := newBufferLogger(LogTrace)
	var target bytes.Buffer

	logger.Print("a")
	logger.Printf("%s", "b")
	logger.Println("c")
	if got, want := selected.String(), "abc\n"; got != want {
		t.Fatalf("selected output = %q, want %q", got, want)
	}

	logger.Fprint(&target, "x")
	logger.Fprintf(&target, "%s", "y")
	logger.Fprintln(&target, "z")
	if got, want := target.String(), "xyz\n"; got != want {
		t.Fatalf("target output = %q, want %q", got, want)
	}
	if got, want := selected.String(), "abc\n"; got != want {
		t.Fatalf("selected output changed to %q, want %q", got, want)
	}
}

func TestPrintErrMethods(t *testing.T) {
	t.Run("stderr only without log file", func(t *testing.T) {
		logger, selected := newBufferLogger(LogAlways)
		gotStderr := captureStderr(t, func() {
			logger.PrintErr("a")
			logger.PrintErrf("%s", "b")
			logger.PrintErrln("c")
		})

		if got, want := gotStderr, "abc\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		if got := selected.String(); got != "" {
			t.Fatalf("selected output = %q, want empty", got)
		}
	})

	t.Run("stderr and log file when configured", func(t *testing.T) {
		logger, path := newFileLogger(t, LogAlways)
		gotStderr := captureStderr(t, func() {
			logger.PrintErr("a")
			logger.PrintErrf("%s", "b")
			logger.PrintErrln("c")
		})

		if got, want := gotStderr, "abc\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		if got, want := readFile(t, path), "abc\n"; got != want {
			t.Fatalf("log content = %q, want %q", got, want)
		}
	})
}

func TestConcurrentPrint(t *testing.T) {
	var buf lockedBuffer
	logger := &Logger{
		Level:          LogTrace,
		output:         &buf,
		ShouldColorize: false,
	}

	const writers = 100
	var wg sync.WaitGroup
	wg.Add(writers)
	for i := 0; i < writers; i++ {
		i := i
		go func() {
			defer wg.Done()
			logger.Print(fmt.Sprintf("message-%03d\n", i))
		}()
	}
	wg.Wait()

	got := buf.String()
	for i := 0; i < writers; i++ {
		assertContains(t, got, fmt.Sprintf("message-%03d\n", i))
	}
}

func TestRichMethods(t *testing.T) {
	block := StructuredTextBlock{Lines: []StyledText{
		{Text: "hello", Style: lipgloss.NewStyle().Bold(true)},
		{Text: " world", Style: lipgloss.NewStyle().Italic(true)},
	}}

	t.Run("Rich raw text when color disabled", func(t *testing.T) {
		logger, buf := newBufferLogger(LogTrace)
		logger.Rich(block)
		if got, want := buf.String(), "hello world"; got != want {
			t.Fatalf("Rich output = %q, want %q", got, want)
		}
	})

	t.Run("Richln appends newline", func(t *testing.T) {
		logger, buf := newBufferLogger(LogTrace)
		logger.Richln(block)
		if got, want := buf.String(), "hello world\n"; got != want {
			t.Fatalf("Richln output = %q, want %q", got, want)
		}
	})

	t.Run("Frich writes target only", func(t *testing.T) {
		logger, selected := newBufferLogger(LogTrace)
		var target bytes.Buffer
		logger.Frich(&target, block)
		if got, want := target.String(), "hello world"; got != want {
			t.Fatalf("Frich target output = %q, want %q", got, want)
		}
		if got := selected.String(); got != "" {
			t.Fatalf("selected output = %q, want empty", got)
		}
	})

	t.Run("non TTY writer gets raw text even when color enabled", func(t *testing.T) {
		var buf bytes.Buffer
		logger := &Logger{Level: LogTrace, output: &buf, ShouldColorize: true}
		logger.Rich(block)
		if got, want := buf.String(), "hello world"; got != want {
			t.Fatalf("Rich output = %q, want %q", got, want)
		}
	})
}

func TestRichErr(t *testing.T) {
	block := StructuredTextBlock{Lines: []StyledText{
		{Text: "Status: ", Style: lipgloss.NewStyle().Bold(true)},
		{Text: "failed", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("9"))},
	}}

	t.Run("stderr only without log file", func(t *testing.T) {
		logger, selected := newBufferLogger(LogAlways)
		gotStderr := captureStderr(t, func() { logger.RichErr(block) })

		if got, want := gotStderr, "Status: failed"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		if got := selected.String(); got != "" {
			t.Fatalf("selected output = %q, want empty", got)
		}
	})

	t.Run("stderr and log file when configured", func(t *testing.T) {
		logger, path := newFileLogger(t, LogAlways)
		gotStderr := captureStderr(t, func() { logger.RichErr(block) })

		if got, want := gotStderr, "Status: failed"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		if got, want := readFile(t, path), "Status: failed"; got != want {
			t.Fatalf("log content = %q, want %q", got, want)
		}
	})
}

func TestCodeMethods(t *testing.T) {
	t.Run("Code indents non-empty lines when color disabled", func(t *testing.T) {
		logger, buf := newBufferLogger(LogTrace)
		logger.Code("first\n\nsecond", "go", "  ")
		if got, want := buf.String(), "  first\n\n  second"; got != want {
			t.Fatalf("Code output = %q, want %q", got, want)
		}
	})

	t.Run("Codeln adds trailing newline", func(t *testing.T) {
		logger, buf := newBufferLogger(LogTrace)
		logger.Codeln("first", "go", "  ")
		if got, want := buf.String(), "  first\n"; got != want {
			t.Fatalf("Codeln output = %q, want %q", got, want)
		}
	})

	t.Run("non TTY writer gets raw indented text even when color enabled", func(t *testing.T) {
		var buf bytes.Buffer
		logger := &Logger{Level: LogTrace, output: &buf, ShouldColorize: true}
		logger.Code("first\nsecond", "go", "\t")
		if got, want := buf.String(), "\tfirst\n\tsecond"; got != want {
			t.Fatalf("Code output = %q, want %q", got, want)
		}
	})
}

func TestLeveledLogging(t *testing.T) {
	tests := []struct {
		name      string
		level     LogLevel
		call      func(*Logger, string)
		prefix    string
		threshold LogLevel
	}{
		{"Error", LogError, func(l *Logger, msg string) { l.Error(msg) }, "ERROR: ", LogError},
		{"Warn", LogWarn, func(l *Logger, msg string) { l.Warn(msg) }, "Warn: ", LogWarn},
		{"Info", LogInfo, func(l *Logger, msg string) { l.Info(msg) }, "INFO: ", LogInfo},
		{"Debug", LogDebug, func(l *Logger, msg string) { l.Debug(msg) }, "DEBUG: ", LogDebug},
		{"Trace", LogTrace, func(l *Logger, msg string) { l.Trace(msg) }, "TRACE: ", LogTrace},
	}

	for _, tt := range tests {
		for level := LogAlways; level <= LogTrace; level++ {
			t.Run(fmt.Sprintf("%s_at_%s", tt.name, level.Name()), func(t *testing.T) {
				logger, path := newFileLogger(t, level)
				msg := "message for " + tt.name

				tt.call(logger, msg)
				got := readFile(t, path)
				if level >= tt.threshold {
					assertContains(t, got, tt.prefix)
					assertContains(t, got, msg)
					assertTimestamp(t, got)
				} else if got != "" {
					t.Fatalf("log content = %q, want empty", got)
				}
			})
		}
	}
}

func TestLeveledFormattingAndNewlineMethods(t *testing.T) {
	tests := []struct {
		name       string
		formatted  func(*Logger)
		newline    func(*Logger)
		wantFormat string
		wantLine   string
	}{
		{
			name:       "Error",
			formatted:  func(l *Logger) { l.Errorf("value=%d", 7) },
			newline:    func(l *Logger) { l.Errorln("line") },
			wantFormat: "value=7",
			wantLine:   "line\n",
		},
		{
			name:       "Warn",
			formatted:  func(l *Logger) { l.Warnf("value=%d", 7) },
			newline:    func(l *Logger) { l.Warnln("line") },
			wantFormat: "value=7",
			wantLine:   "line\n",
		},
		{
			name:       "Info",
			formatted:  func(l *Logger) { l.Infof("value=%d", 7) },
			newline:    func(l *Logger) { l.Infoln("line") },
			wantFormat: "value=7",
			wantLine:   "line\n",
		},
		{
			name:       "Debug",
			formatted:  func(l *Logger) { l.Debugf("value=%d", 7) },
			newline:    func(l *Logger) { l.Debugln("line") },
			wantFormat: "value=7",
			wantLine:   "line\n",
		},
		{
			name:       "Trace",
			formatted:  func(l *Logger) { l.Tracef("value=%d", 7) },
			newline:    func(l *Logger) { l.Traceln("line") },
			wantFormat: "value=7",
			wantLine:   "line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, path := newFileLogger(t, LogTrace)
			tt.formatted(logger)
			tt.newline(logger)

			got := readFile(t, path)
			assertContains(t, got, tt.wantFormat)
			assertContains(t, got, tt.wantLine)
		})
	}
}

func TestRoutingAndGetErrorPipe(t *testing.T) {
	t.Run("file backed logger routes selected levels to file", func(t *testing.T) {
		logger, path := newFileLogger(t, LogTrace)

		logger.Info("info")
		logger.Debug("debug")
		logger.Trace("trace")
		logger.Error("error")
		logger.Warn("warn")

		got := readFile(t, path)
		for _, want := range []string{"INFO: ", "info", "DEBUG: ", "debug", "TRACE: ", "trace", "ERROR: ", "error", "Warn: ", "warn"} {
			assertContains(t, got, want)
		}
	})

	t.Run("GetErrorPipe returns file when present", func(t *testing.T) {
		logger, _ := newFileLogger(t, LogInfo)
		if got := logger.GetErrorPipe(); got != logger.outputFile {
			t.Fatalf("GetErrorPipe() = %v, want outputFile", got)
		}
	})

	t.Run("GetErrorPipe returns stderr without file", func(t *testing.T) {
		logger, _ := newBufferLogger(LogInfo)
		if got := logger.GetErrorPipe(); got != os.Stderr {
			t.Fatalf("GetErrorPipe() = %v, want os.Stderr", got)
		}
	})
}

func TestPanicMethods(t *testing.T) {
	t.Run("Panic below debug", func(t *testing.T) {
		logger, path := newFileLogger(t, LogInfo)
		got := mustPanicLoggerPanic(t, func() { logger.Panic("boom") })

		if got.Message != "boom" {
			t.Fatalf("panic message = %q, want %q", got.Message, "boom")
		}
		content := readFile(t, path)
		assertContains(t, content, "Error: ")
		assertContains(t, content, "boom")
		assertNotContains(t, content, "PANIC: ")
	})

	t.Run("Panic at debug includes panic prefix and timestamp", func(t *testing.T) {
		logger, path := newFileLogger(t, LogDebug)
		got := mustPanicLoggerPanic(t, func() { logger.Panic("boom") })

		if got.Message != "boom" {
			t.Fatalf("panic message = %q, want %q", got.Message, "boom")
		}
		content := readFile(t, path)
		assertContains(t, content, "PANIC: ")
		assertContains(t, content, "boom")
		assertTimestamp(t, content)
	})

	t.Run("Panicf formats message", func(t *testing.T) {
		logger, _ := newFileLogger(t, LogDebug)
		got := mustPanicLoggerPanic(t, func() { logger.Panicf("boom %d", 7) })
		if got.Message != "boom 7" {
			t.Fatalf("panic message = %q, want %q", got.Message, "boom 7")
		}
	})

	t.Run("Panicln includes newline", func(t *testing.T) {
		logger, _ := newFileLogger(t, LogDebug)
		got := mustPanicLoggerPanic(t, func() { logger.Panicln("boom") })
		if got.Message != "boom\n" {
			t.Fatalf("panic message = %q, want %q", got.Message, "boom\n")
		}
	})
}

func TestFatalMethods(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		want     string
	}{
		{"Fatal", 11, "fatal message"},
		{"Fatalf", 12, "fatal 7"},
		{"Fatalln", 13, "fatal line\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logPath := filepath.Join(t.TempDir(), "fatal.log")
			cmd := exec.Command(os.Args[0], "-test.run=TestFatalSubprocess")
			cmd.Env = append(os.Environ(),
				"CLILOG_FATAL_TEST="+tt.name,
				"CLILOG_FATAL_PATH="+logPath,
			)

			err := cmd.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("subprocess error = %v, want ExitError", err)
			}
			if got := exitErr.ExitCode(); got != tt.exitCode {
				t.Fatalf("exit code = %d, want %d", got, tt.exitCode)
			}

			content := readFile(t, logPath)
			assertContains(t, content, "FATAL: ")
			assertContains(t, content, tt.want)
			assertTimestamp(t, content)
		})
	}
}

func TestFatalSubprocess(t *testing.T) {
	mode := os.Getenv("CLILOG_FATAL_TEST")
	if mode == "" {
		t.Skip("fatal subprocess helper")
	}

	path := os.Getenv("CLILOG_FATAL_PATH")
	if path == "" {
		t.Fatal("CLILOG_FATAL_PATH is empty")
	}

	logger := NewLogger(int(LogTrace), path)
	logger.ShouldColorize = false
	switch mode {
	case "Fatal":
		logger.Fatal(11, "fatal message")
	case "Fatalf":
		logger.Fatalf(12, "fatal %d", 7)
	case "Fatalln":
		logger.Fatalln(13, "fatal line")
	default:
		t.Fatalf("unknown fatal mode %q", mode)
	}
}

func TestPackageLevelHelpers(t *testing.T) {
	logger, selected := newBufferLogger(LogTrace)
	withGlobalLogger(t, logger)

	Print("a")
	Printf("%s", "b")
	Println("c")
	if got, want := selected.String(), "abc\n"; got != want {
		t.Fatalf("global selected output = %q, want %q", got, want)
	}

	var target bytes.Buffer
	Fprint(&target, "x")
	Fprintf(&target, "%s", "y")
	Fprintln(&target, "z")
	if got, want := target.String(), "xyz\n"; got != want {
		t.Fatalf("global target output = %q, want %q", got, want)
	}

	Rich(StructuredTextBlock{Lines: []StyledText{{Text: "rich"}}})
	Richln(StructuredTextBlock{Lines: []StyledText{{Text: "line"}}})
	var richTarget bytes.Buffer
	Frich(&richTarget, StructuredTextBlock{Lines: []StyledText{{Text: "frich"}}})
	Code("code", "go", "> ")
	Codeln("codeln", "go", "> ")

	gotSelected := selected.String()
	for _, want := range []string{"rich", "line\n", "> code", "> codeln\n"} {
		assertContains(t, gotSelected, want)
	}
	if got, want := richTarget.String(), "frich"; got != want {
		t.Fatalf("Frich output = %q, want %q", got, want)
	}

	gotStderr := captureStderr(t, func() {
		PrintErr("plain ")
		PrintErrf("%s", "formatted ")
		PrintErrln("line")
		RichErr(StructuredTextBlock{Lines: []StyledText{{Text: "rich error"}}})
	})
	if got, want := gotStderr, "plain formatted line\nrich error"; got != want {
		t.Fatalf("global stderr = %q, want %q", got, want)
	}

	Error("error")
	Warn("warn")
	Info("info")
	Debug("debug")
	Trace("trace")

	gotSelected = selected.String()
	for _, want := range []string{"INFO: ", "info", "DEBUG: ", "debug", "TRACE: ", "trace"} {
		assertContains(t, gotSelected, want)
	}
	if got := GetErrorPipe(); got != os.Stderr {
		t.Fatalf("GetErrorPipe() = %v, want os.Stderr", got)
	}
}

func TestPackageLevelPanicHelpers(t *testing.T) {
	logger, _ := newBufferLogger(LogTrace)
	withGlobalLogger(t, logger)

	if got := mustPanicLoggerPanic(t, func() { Panic("boom") }); got.Message != "boom" {
		t.Fatalf("Panic message = %q, want %q", got.Message, "boom")
	}
	if got := mustPanicLoggerPanic(t, func() { Panicf("boom %d", 7) }); got.Message != "boom 7" {
		t.Fatalf("Panicf message = %q, want %q", got.Message, "boom 7")
	}
	if got := mustPanicLoggerPanic(t, func() { Panicln("boom") }); got.Message != "boom\n" {
		t.Fatalf("Panicln message = %q, want %q", got.Message, "boom\n")
	}
}
