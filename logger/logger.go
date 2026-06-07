package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
}

// Subscriber receives new log entries in real-time via a channel.
type subscriber struct {
	ch chan LogEntry
}

// Logger provides asynchronous logging to file with a ring buffer for GUI access.
type Logger struct {
	ch          chan LogEntry
	file        *os.File
	ring        *RingBuffer
	done        chan struct{}
	wg          sync.WaitGroup
	subscribers []subscriber
	subMu       sync.Mutex
}

// NewLogger creates a new async logger writing to the given file path.
func NewLogger(filePath string, ringSize int) (*Logger, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot open log file: %w", err)
	}

	l := &Logger{
		ch:   make(chan LogEntry, 1024),
		file: file,
		ring: NewRingBuffer(ringSize),
		done: make(chan struct{}),
	}

	l.wg.Add(1)
	go l.run()
	return l, nil
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   fmt.Sprintf(format, args...),
	}
	select {
	case l.ch <- entry:
	default:
		// channel full - drop the log entry (non-blocking)
	}
}

// Ring returns the underlying ring buffer for GUI access.
func (l *Logger) Ring() *RingBuffer {
	return l.ring
}

// Subscribe returns a channel that receives every new log entry in real-time.
// The caller should read from the channel promptly; slow readers may miss entries.
// Call Unsubscribe with the returned channel to stop receiving.
func (l *Logger) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, 256)
	l.subMu.Lock()
	l.subscribers = append(l.subscribers, subscriber{ch: ch})
	l.subMu.Unlock()
	return ch
}

// Unsubscribe removes a previously subscribed channel.
func (l *Logger) Unsubscribe(ch chan LogEntry) {
	l.subMu.Lock()
	defer l.subMu.Unlock()
	for i, s := range l.subscribers {
		if s.ch == ch {
			l.subscribers = append(l.subscribers[:i], l.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// Close shuts down the logger and waits for pending writes.
func (l *Logger) Close() error {
	close(l.done)
	l.wg.Wait()
	return l.file.Close()
}

func (l *Logger) run() {
	defer l.wg.Done()
	for {
		select {
		case entry := <-l.ch:
			l.ring.Add(entry)
			_, _ = fmt.Fprintf(l.file, "[%s] %s %s\n",
				entry.Timestamp.Format("2006-01-02 15:04:05"),
				entry.Level,
				entry.Message,
			)
			// Notify subscribers (non-blocking — drop if slow)
			l.subMu.Lock()
			for _, s := range l.subscribers {
				select {
				case s.ch <- entry:
				default:
				}
			}
			l.subMu.Unlock()
		case <-l.done:
			// Drain remaining entries
			for {
				select {
				case entry := <-l.ch:
					l.ring.Add(entry)
					_, _ = fmt.Fprintf(l.file, "[%s] %s %s\n",
						entry.Timestamp.Format("2006-01-02 15:04:05"),
						entry.Level,
						entry.Message,
					)
					// Notify subscribers during drain too
					l.subMu.Lock()
					for _, s := range l.subscribers {
						select {
						case s.ch <- entry:
						default:
						}
					}
					l.subMu.Unlock()
				default:
					return
				}
			}
		}
	}
}
