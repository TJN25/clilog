# clilog
clilog is a stylized, thread-safe logger for Go CLI tools. It combines structured logging levels with rich terminal styling (via Lip Gloss) and syntax highlighting (via Chroma).
Features
Leveled Logging: Support for Error, Warn, Info, Debug, and Trace.
Intelligent Routing:
Errors/Warnings: Behave like a "Tee" (written to both the log file and stderr).
Info/Debug: Behave like a "Redirect" (written to the log file if present, otherwise stdout).
Syntax Highlighting: dedicated Code() method for printing syntax-highlighted code snippets to the terminal.
TTY Detection: Automatically strips ANSI color codes when writing to files, while keeping them for the terminal.
Thread Safe: Built-in mutex locking for concurrent file writes.
Installation
go get [github.com/yourusername/clilog](https://github.com/yourusername/clilog)


Usage
Initialization
package main

import "[github.com/yourusername/clilog](https://github.com/yourusername/clilog)"

func main() {
    // Level 4 = Debug
    // "app.log" = Output file (optional, pass "" or "stdout" to skip)
    log := clilog.NewLogger(4, "app.log")
    defer log.Close()

    log.Info("Application started")
}


Logging Levels
log.Error("Something went wrong!") // Prints to stderr AND file
log.Warn("Disk is filling up")     // Prints to stderr AND file
log.Info("Normal operation")       // Prints to file (or stdout if no file)
log.Debug("Variable x = 42")       // Prints to file (or stdout if no file)


Printing Code
The Code method uses Chroma to highlight code snippets. Note that this always writes to stdout to preserve formatting for the user.
jsonConfig := `{"foo": "bar", "baz": 123}`
log.Code(jsonConfig, "json", "  ") // Message, Language, Indentation


Styling
The logger comes with pre-defined Lip Gloss styles for different log levels:
Error: Red & Bold
Warn: Yellow
Info: Green
Debug: Blue
Trace: Red & Bold

