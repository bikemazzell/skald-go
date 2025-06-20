package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ValidationModeSecurityFocused = "security_focused"
	ValidationModeDisabled        = "disabled"
)

type WhisperModelInfo struct {
	URL    string `json:"url"`
	Size   string `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
}

type AudioConfig struct {
	SampleRate           int        `json:"sample_rate"`
	Channels             int        `json:"channels"`
	SilenceThreshold     float32    `json:"silence_threshold"`
	SilenceDuration      float32    `json:"silence_duration"`
	ChunkDuration        int        `json:"chunk_duration"`
	MaxDuration          int        `json:"max_duration"`
	BufferSizeMultiplier int        `json:"buffer_size_multiplier"`
	FrameLength          int        `json:"frame_length"`
	BufferedFrames       int        `json:"buffered_frames"`
	DeviceIndex          int        `json:"device_index"`
	StartTone            ToneConfig `json:"start_tone"`
	CompletionTone       ToneConfig `json:"completion_tone"`
	ErrorTone            ToneConfig `json:"error_tone"`
}

type ToneConfig struct {
	Enabled   bool `json:"enabled"`
	Frequency int  `json:"frequency"`
	Duration  int  `json:"duration"`
	FadeMs    int  `json:"fade_ms"`
}

type ContinuousModeConfig struct {
	Enabled            bool `json:"enabled"`
	MaxSessionDuration int  `json:"max_session_duration"`
	InterSpeechTimeout int  `json:"inter_speech_timeout"`
	AutoStopOnIdle     bool `json:"auto_stop_on_idle"`
}

type TextValidationConfig struct {
	Mode             string   `json:"mode"`
	AllowPunctuation bool     `json:"allow_punctuation"`
	CustomBlocklist  []string `json:"custom_blocklist"`
}

type ProcessingConfig struct {
	ShutdownTimeout   int                  `json:"shutdown_timeout"`
	EventWaitTimeout  float64              `json:"event_wait_timeout"`
	AutoPaste         bool                 `json:"auto_paste"`
	ChannelBufferSize int                  `json:"channel_buffer_size"`
	ContinuousMode    ContinuousModeConfig `json:"continuous_mode"`
	TextValidation    TextValidationConfig `json:"text_validation"`
}

type WhisperConfig struct {
	Model              string                      `json:"model"`
	Language           string                      `json:"language"`
	AutoDetectLanguage bool                        `json:"auto_detect_language"`
	SupportedLanguages []string                    `json:"supported_languages,omitempty"`
	BeamSize           int                         `json:"beam_size"`
	Models             map[string]WhisperModelInfo `json:"models"`
}

type ServerConfig struct {
	SocketPath      string            `json:"socket_path"`
	SocketTimeout   float32           `json:"socket_timeout"`
	KeyboardEnabled bool              `json:"keyboard_enabled"`
	Hotkeys         map[string]string `json:"hotkeys"`
}

type DebugConfig struct {
	PrintStatus         bool `json:"print_status"`
	PrintTranscriptions bool `json:"print_transcriptions"`
}

type Config struct {
	Version    string           `json:"version"`
	Verbose    bool             `json:"-"`
	Audio      AudioConfig      `json:"audio"`
	Processing ProcessingConfig `json:"processing"`
	Whisper    WhisperConfig    `json:"whisper"`
	Server     ServerConfig     `json:"server"`
	Debug      DebugConfig      `json:"debug"`
}

func DefaultAudioConfig() AudioConfig {
	return AudioConfig{
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
		StartTone: ToneConfig{
			Enabled:   true,
			Frequency: 440,
			Duration:  150,
			FadeMs:    5,
		},
		CompletionTone: ToneConfig{
			Enabled:   true,
			Frequency: 660,
			Duration:  200,
			FadeMs:    10,
		},
		ErrorTone: ToneConfig{
			Enabled:   true,
			Frequency: 220,
			Duration:  300,
			FadeMs:    15,
		},
	}
}

func DefaultProcessingConfig() ProcessingConfig {
	return ProcessingConfig{
		ShutdownTimeout:   30,
		EventWaitTimeout:  0.1,
		AutoPaste:         true,
		ChannelBufferSize: 10,
		ContinuousMode: ContinuousModeConfig{
			Enabled:            true,
			MaxSessionDuration: 300,
			InterSpeechTimeout: 10,
			AutoStopOnIdle:     true,
		},
		TextValidation: TextValidationConfig{
			Mode:             ValidationModeSecurityFocused,
			AllowPunctuation: true,
			CustomBlocklist:  []string{},
		},
	}
}

func DefaultWhisperConfig() WhisperConfig {
	return WhisperConfig{
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
	}
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
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
	}
}

func DefaultDebugConfig() DebugConfig {
	return DebugConfig{
		PrintStatus:         true,
		PrintTranscriptions: true,
	}
}

func DefaultConfig() *Config {
	return &Config{
		Version:    "0.1",
		Audio:      DefaultAudioConfig(),
		Processing: DefaultProcessingConfig(),
		Whisper:    DefaultWhisperConfig(),
		Server:     DefaultServerConfig(),
		Debug:      DefaultDebugConfig(),
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
	if err := c.validateBasicSettings(); err != nil {
		return err
	}
	if err := c.Whisper.Validate(c.Audio.SampleRate); err != nil {
		return err
	}
	if err := c.Audio.Validate(); err != nil {
		return err
	}
	if err := c.Processing.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateBasicSettings() error {
	if c.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	return nil
}

func (wc *WhisperConfig) Validate(sampleRate int) error {
	if wc.Model == "" {
		return fmt.Errorf("whisper model cannot be empty")
	}
	if _, ok := wc.Models[wc.Model]; !ok {
		return fmt.Errorf("model %q not found in configuration", wc.Model)
	}

	if sampleRate != 16000 {
		return fmt.Errorf("whisper models require a sample rate of 16000, but got %d", sampleRate)
	}

	if wc.Language == "" && !wc.AutoDetectLanguage {
		return fmt.Errorf("language must be specified if auto-detect is disabled")
	}

	if wc.BeamSize <= 0 {
		return fmt.Errorf("beam_size must be positive")
	}

	return nil
}

func (ac *AudioConfig) Validate() error {
	if ac.SampleRate <= 0 {
		return fmt.Errorf("sample_rate must be positive")
	}
	if ac.Channels <= 0 {
		return fmt.Errorf("channels must be positive")
	}
	if ac.ChunkDuration <= 0 {
		return fmt.Errorf("chunk_duration must be positive")
	}
	if ac.MaxDuration <= 0 {
		return fmt.Errorf("max_duration must be positive")
	}
	return nil
}

func (pc *ProcessingConfig) Validate() error {
	if pc.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown_timeout must be positive")
	}
	if pc.EventWaitTimeout <= 0 {
		return fmt.Errorf("event_wait_timeout must be positive")
	}
	if pc.ChannelBufferSize <= 0 {
		return fmt.Errorf("channel_buffer_size must be positive")
	}
	return nil
}
