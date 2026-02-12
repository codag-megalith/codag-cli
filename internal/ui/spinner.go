package ui

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

var brailleFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var asciiFrames = []string{"|", "/", "-", "\\"}

type Spinner struct {
	mu      sync.Mutex
	message string
	done    chan struct{}
	frames  []string
}

func NewSpinner(message string) *Spinner {
	frames := brailleFrames
	if runtime.GOOS == "windows" {
		frames = asciiFrames
	}
	return &Spinner{
		message: message,
		done:    make(chan struct{}),
		frames:  frames,
	}
}

func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				// Clear the spinner line
				fmt.Print("\r\033[2K")
				return
			default:
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				frame := Cyan.Render(s.frames[i%len(s.frames)])
				fmt.Printf("\r\033[2K%s %s", frame, msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Update(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

func (s *Spinner) Stop() {
	select {
	case <-s.done:
		// Already stopped
	default:
		close(s.done)
		// Give the goroutine a moment to clear the line
		time.Sleep(100 * time.Millisecond)
	}
}
