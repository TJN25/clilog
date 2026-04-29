# clilog

`clilog` is a small logger for Go CLI tools. It provides leveled output, optional file logging, terminal styling with Lip Gloss, and syntax-highlighted code blocks with Chroma.

It is designed for command-line tools that want readable terminal output without giving up simple log-file routing.

## Features

- **Leveled logging:** `Error`, `Warn`, `Info`, `Debug`, and `Trace`
- **Global logger:** use `clilog.Infof(...)` directly after initialization
- **Explicit logger instances:** use `clilog.NewLogger(...)` when you want separate loggers
- **Output routing:**
  - `Error`, `Warn`, `Panic`, and `Fatal` write to `stderr`
  - if a log file is configured, those messages are also written to the file
  - `Info`, `Debug`, `Trace`, `Print`, `Rich`, and `Code` write to the selected logger output
- **Terminal styling:** uses Lip Gloss styles for readable CLI output
- **TTY-aware output:** avoids ANSI styling when writing to files or non-terminal targets
- **Syntax highlighting:** `Code()` renders highlighted code when writing to a terminal
- **Thread-safe writes:** logger writes are protected by a mutex

## Installation

```bash
go get github.com/TJN25/clilog
```

## Basic usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/TJN25/clilog"
)

func main() {
	if err := clilog.InitializeLogger(3, "app.log"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer clilog.CloseLogger()

	clilog.Info("Application started")
	clilog.Warn("Disk space is getting low")
}
```

## Logging levels

Verbosity controls which leveled messages are written.

| Verbosity | Enabled levels |
|---:|---|
| `0` | manual output only, such as `Print` |
| `1` | `Error` |
| `2` | `Error`, `Warn` |
| `3` | `Error`, `Warn`, `Info` |
| `4` | `Error`, `Warn`, `Info`, `Debug` |
| `5` | `Error`, `Warn`, `Info`, `Debug`, `Trace` |

```go
clilog.Error("Something went wrong")
clilog.Warn("Disk is filling up")
clilog.Info("Normal operation")
clilog.Debug("Variable x = 42")
clilog.Trace("Detailed execution path")
```

Formatted variants are also available:

```go
clilog.Infof("installed %s@%s", slug, version)
clilog.Errorf("failed to open %s: %v", path, err)
```

## Global logger

The package provides a global logger, initialized to write to stdout by default.

```go
clilog.Info("uses the package global logger")
```

To configure it:

```go
err := clilog.InitializeLogger(4, "app.log")
if err != nil {
	// handle error from closing any previous logger
}
defer clilog.CloseLogger()
```

Passing `"stdout"` or `""` leaves output on stdout:

```go
clilog.InitializeLogger(4, "stdout")
clilog.InitializeLogger(4, "")
```

## Explicit logger instances

You can also create a logger directly:

```go
log := clilog.NewLogger(4, "app.log")
defer log.Close()

log.Info("Application started")
log.Debugf("config path: %s", configPath)
```

This is useful when a project wants to manage logger instances itself instead of using the package global logger.

## Routing behavior

When no log file is configured, normal output goes to stdout.

When a log file is configured, normal output goes to the file.

Errors and warnings are treated specially: they are written to stderr, and also written to the log file when one is configured.

```go
clilog.Info("written to selected output")
clilog.Error("written to stderr, and also to the log file if configured")
```

For explicit targets, use the `Fprint` family:

```go
clilog.Fprintln(os.Stderr, "message to stderr")
```

## Rich output

`Rich` accepts styled text blocks and writes them as a single output operation.

```go
block := clilog.StructuredTextBlock{
	Lines: []clilog.StyledText{
		{Text: "Status: ", Style: clilog.BlueBoldStyle},
		{Text: "OK\n", Style: clilog.GreenStyle},
	},
}

clilog.Rich(block)
```

`Frich` writes a styled block to an explicit target:

```go
clilog.Frich(os.Stderr, block)
```

## Printing code

`Code` prints syntax-highlighted code when writing to a terminal. When writing to a file or non-terminal target, it writes plain text.

```go
jsonConfig := `{"foo": "bar", "baz": 123}`

clilog.Code(jsonConfig, "json", "  ")
```

Arguments are:

```go
clilog.Code(message, language, indent)
```

The syntax-highlighting formatter and style are stored on the logger:

```go
log := clilog.NewLogger(4, "stdout")
log.CodeStyle.Formatter = "terminal256"
log.CodeStyle.Style = "catppuccin-latte"
```

## Panics and fatal errors

`Panic` logs the message, then panics with `clilog.LoggerPanic`.

```go
clilog.Panic("unrecoverable logger-controlled panic")
```

`Fatal` logs the message, then exits with the supplied exit code.

```go
clilog.Fatal(1, "cannot continue")
```

The package defines `LoggerPanic`, but applications should decide for themselves whether and how to recover from it.

## Styles

`clilog` exposes the default Lip Gloss styles used by the logger:

```go
clilog.ErrorStyle
clilog.WarningStyle
clilog.InfoStyle
clilog.DebugStyle
clilog.TraceStyle
clilog.TimestampStyle
clilog.MessageDefaultStyle
```

Additional color helpers are also available:

```go
clilog.RedStyle
clilog.YellowStyle
clilog.GreenStyle
clilog.BlueStyle
clilog.BlueBoldStyle
clilog.BlueItalicStyle
```
