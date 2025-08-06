module skald

go 1.23.0

toolchain go1.24.4

require (
	github.com/gen2brain/malgo v0.11.23
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20250802050304-0becabc8d68d
)

replace github.com/ggerganov/whisper.cpp/bindings/go => ./deps/whisper-go

replace github.com/ggerganov/whisper.cpp => ./deps/whisper.cpp
