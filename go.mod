module skald

go 1.23

toolchain go1.23.3

require (
	github.com/atotto/clipboard v0.1.4
	github.com/gen2brain/malgo v0.11.23
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20241121150429-8c6a9b8bb6a0
)

replace github.com/ggerganov/whisper.cpp => ./external/whisper.cpp
