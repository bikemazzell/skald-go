package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type WhisperModelInfo struct {
	URL      string `json:"url"`
	Size     string `json:"size"`
	SHA256   string `json:"sha256,omitempty"`
}

type Config struct {
	Version string `json:"version"`
	Verbose bool   `json:"-"`
	Audio   struct {
		SampleRate           int     `json:"sample_rate"`
		Channels             int     `json:"channels"`
		SilenceThreshold     float32 `json:"silence_threshold"`
		SilenceDuration      float32 `json:"silence_duration"`
		ChunkDuration        int     `json:"chunk_duration"`
		MaxDuration          int     `json:"max_duration"`
		BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
		FrameLength          int     `json:"frame_length"`
		BufferedFrames       int     `json:"buffered_frames"`
		DeviceIndex          int     `json:"device_index"`
		StartTone            struct {
			Enabled   bool `json:"enabled"`
			Frequency int  `json:"frequency"`
			Duration  int  `json:"duration"`
			FadeMs    int  `json:"fade_ms"`
		} `json:"start_tone"`
		CompletionTone struct {
			Enabled   bool `json:"enabled"`
			Frequency int  `json:"frequency"`
			Duration  int  `json:"duration"`
			FadeMs    int  `json:"fade_ms"`
		} `json:"completion_tone"`
		ErrorTone struct {
			Enabled   bool `json:"enabled"`
			Frequency int  `json:"frequency"`
			Duration  int  `json:"duration"`
			FadeMs    int  `json:"fade_ms"`
		} `json:"error_tone"`
	} `json:"audio"`
	Processing struct {
		ShutdownTimeout   int     `json:"shutdown_timeout"`
		EventWaitTimeout  float64 `json:"event_wait_timeout"`
		AutoPaste         bool    `json:"auto_paste"`
		ChannelBufferSize int     `json:"channel_buffer_size"`
		ContinuousMode    struct {
			Enabled              bool `json:"enabled"`
			MaxSessionDuration   int  `json:"max_session_duration"`
			InterSpeechTimeout   int  `json:"inter_speech_timeout"`
			AutoStopOnIdle       bool `json:"auto_stop_on_idle"`
		} `json:"continuous_mode"`
		TextValidation struct {
			Mode               string   `json:"mode"`
			AllowPunctuation   bool     `json:"allow_punctuation"`
			CustomBlocklist    []string `json:"custom_blocklist"`
		} `json:"text_validation"`
	} `json:"processing"`
	Whisper struct {
		Model              string                      `json:"model"`
		Language           string                      `json:"language"`
		AutoDetectLanguage bool                        `json:"auto_detect_language"`
		SupportedLanguages []string                    `json:"supported_languages,omitempty"`
		BeamSize           int                         `json:"beam_size"`
		Models             map[string]WhisperModelInfo `json:"models"`
	} `json:"whisper"`
	Server struct {
		SocketPath      string             `json:"socket_path"`
		SocketTimeout   float32            `json:"socket_timeout"`
		KeyboardEnabled bool               `json:"keyboard_enabled"`
		Hotkeys         map[string]string  `json:"hotkeys"`
	} `json:"server"`
	Debug struct {
		PrintStatus         bool `json:"print_status"`
		PrintTranscriptions bool `json:"print_transcriptions"`
	} `json:"debug"`
}

func DefaultConfig() *Config {
	return &Config{
		Version: "0.1",
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
			CompletionTone struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"completion_tone"`
			ErrorTone struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"error_tone"`
		}{
			SampleRate:           16000,
			Channels:             1,
			SilenceThreshold:     0.008,
			SilenceDuration:      3.0,
			ChunkDuration:        30,
			MaxDuration:          300,
			BufferSizeMultiplier: 2,
			FrameLength:          512,
			BufferedFrames:       10,
			DeviceIndex:          -1,
			StartTone: struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			}{
				Enabled:   true,
				Frequency: 440,
				Duration:  150,
				FadeMs:    5,
			},
			CompletionTone: struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			}{
				Enabled:   true,
				Frequency: 660,
				Duration:  200,
				FadeMs:    10,
			},
			ErrorTone: struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			}{
				Enabled:   true,
				Frequency: 220,
				Duration:  300,
				FadeMs:    15,
			},
		},
		Processing: struct {
			ShutdownTimeout   int     `json:"shutdown_timeout"`
			EventWaitTimeout  float64 `json:"event_wait_timeout"`
			AutoPaste         bool    `json:"auto_paste"`
			ChannelBufferSize int     `json:"channel_buffer_size"`
			ContinuousMode    struct {
				Enabled              bool `json:"enabled"`
				MaxSessionDuration   int  `json:"max_session_duration"`
				InterSpeechTimeout   int  `json:"inter_speech_timeout"`
				AutoStopOnIdle       bool `json:"auto_stop_on_idle"`
			} `json:"continuous_mode"`
			TextValidation struct {
				Mode               string   `json:"mode"`
				AllowPunctuation   bool     `json:"allow_punctuation"`
				CustomBlocklist    []string `json:"custom_blocklist"`
			} `json:"text_validation"`
		}{
			ShutdownTimeout:   30,
			EventWaitTimeout:  0.1,
			AutoPaste:         true,
			ChannelBufferSize: 10,
			ContinuousMode: struct {
				Enabled              bool `json:"enabled"`
				MaxSessionDuration   int  `json:"max_session_duration"`
				InterSpeechTimeout   int  `json:"inter_speech_timeout"`
				AutoStopOnIdle       bool `json:"auto_stop_on_idle"`
			}{
				Enabled:              true,
				MaxSessionDuration:   300,
				InterSpeechTimeout:   10,
				AutoStopOnIdle:       true,
			},
			TextValidation: struct {
				Mode               string   `json:"mode"`
				AllowPunctuation   bool     `json:"allow_punctuation"`
				CustomBlocklist    []string `json:"custom_blocklist"`
			}{
				Mode:               "security_focused",
				AllowPunctuation:   true,
				CustomBlocklist:    []string{},
			},
		},
		Whisper: struct {
			Model              string                      `json:"model"`
			Language           string                      `json:"language"`
			AutoDetectLanguage bool                        `json:"auto_detect_language"`
			SupportedLanguages []string                    `json:"supported_languages,omitempty"`
			BeamSize           int                         `json:"beam_size"`
				Models             map[string]WhisperModelInfo `json:"models"`
		}{
			Model:              "large-v3-turbo-q8_0",
			Language:           "en",
			AutoDetectLanguage: false,
			SupportedLanguages: []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"},
			BeamSize:           5,
			Models: map[string]WhisperModelInfo{
				"tiny.en": {
					URL:  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin",
					Size: "77.7MB",
				},
				"large-v3-turbo-q8_0": {
					URL:  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q8_0.bin",
					Size: "874MB",
				},
			},
		},
		Server: struct {
			SocketPath      string             `json:"socket_path"`
			SocketTimeout   float32            `json:"socket_timeout"`
			KeyboardEnabled bool               `json:"keyboard_enabled"`
			Hotkeys         map[string]string  `json:"hotkeys"`
		}{
			SocketPath:      "/tmp/skald.sock",
			SocketTimeout:   5.0,
			KeyboardEnabled: true,
			Hotkeys: map[string]string{
				"r": "start",
				"s": "stop", 
				"i": "status",
				"q": "quit",
				"?": "help",
				"c": "resume",
			},
		},
		Debug: struct {
			PrintStatus         bool `json:"print_status"`
			PrintTranscriptions bool `json:"print_transcriptions"`
		}{
			PrintStatus:         true,
			PrintTranscriptions: true,
		},
	}
}

func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if err := Save(absPath, cfg); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0640); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	if err := c.validateBasicSettings(); err != nil {
		return err
	}

	if err := c.validateAudioSettings(); err != nil {
		return err
	}

	if err := c.validateProcessingSettings(); err != nil {
		return err
	}

	if err := c.validateWhisperSettings(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateBasicSettings() error {
	if c.Server.SocketPath == "" {
		return fmt.Errorf("server.socket_path cannot be empty")
	}
	if c.Server.SocketTimeout <= 0 {
		return fmt.Errorf("server.socket_timeout must be positive")
	}
	return nil
}

func (c *Config) validateWhisperSettings() error {
	modelInfo, exists := c.Whisper.Models[c.Whisper.Model]
	if !exists {
		return fmt.Errorf("model '%s' not found in models configuration", c.Whisper.Model)
	}

	if modelInfo.URL == "" {
		return fmt.Errorf("URL for model %s cannot be empty", c.Whisper.Model)
	}
	if modelInfo.Size == "" {
		return fmt.Errorf("size for model %s cannot be empty", c.Whisper.Model)
	}

	if c.Whisper.BeamSize <= 0 {
		return fmt.Errorf("beam size must be positive")
	}

	if c.Whisper.Language == "" {
		return fmt.Errorf("whisper language cannot be empty")
	}

	return nil
}

func (c *Config) validateAudioSettings() error {
	if c.Audio.SampleRate <= 0 {
		return fmt.Errorf("sample rate must be positive")
	}
	if c.Audio.Channels <= 0 {
		return fmt.Errorf("channels must be positive")
	}
	if c.Audio.SilenceThreshold < 0 || c.Audio.SilenceThreshold > 1 {
		return fmt.Errorf("silence threshold must be between 0 and 1")
	}
	if c.Audio.SilenceDuration <= 0 {
		return fmt.Errorf("silence duration must be positive")
	}
	if c.Audio.FrameLength <= 0 {
		return fmt.Errorf("frame length must be positive")
	}
	if c.Audio.BufferedFrames <= 0 {
		return fmt.Errorf("buffered frames must be positive")
	}
	if c.Audio.DeviceIndex < -1 {
		return fmt.Errorf("device index must be -1 or greater")
	}
	return nil
}

func (c *Config) validateProcessingSettings() error {
	if c.Processing.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}
	if c.Processing.ChannelBufferSize <= 0 {
		return fmt.Errorf("channel buffer size must be positive")
	}
	return nil
}
