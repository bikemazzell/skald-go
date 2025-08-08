module skald

go 1.23.0

toolchain go1.24.4

require (
	github.com/gen2brain/malgo v0.11.23
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20250807023745-4245c77b654c
)

replace github.com/ggerganov/whisper.cpp/bindings/go => ./deps/whisper-go

replace github.com/ggerganov/whisper.cpp => ./deps/whisper.cpp
