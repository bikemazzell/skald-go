package server

import (
	"fmt"

	"github.com/eiannone/keyboard"
)

func (s *Server) startKeyboardListener() {
	s.mu.Lock()
	s.keyboardActive = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.keyboardActive = false
		s.mu.Unlock()
	}()

	if err := keyboard.Open(); err != nil {
		s.logger.Printf("Failed to open keyboard: %v", err)
		return
	}
	defer keyboard.Close()

	fmt.Println("Keyboard listener started. Press '?' for help.")

	for {
		select {
		case <-s.keyboardCtx.Done():
			return
		default:
			char, key, err := keyboard.GetKey()
			if err != nil {
				s.logger.Printf("Error getting key: %v", err)
				return
			}
			if key == keyboard.KeyEsc || key == keyboard.KeyCtrlC {
				s.handleQuit()
				return
			}
			s.handleKeyPress(char)
		}
	}
}

func (s *Server) handleKeyPress(key rune) {
	action, exists := s.keyActions[key]
	if exists {
		if s.cfg.Verbose {
			s.logger.Printf("Executing keyboard action: %s", action.Desc)
		}
		if err := action.Handler(); err != nil {
			s.logger.Printf("Error executing action: %v", err)
		}
	}
}
