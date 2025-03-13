package main

// #cgo CFLAGS: -I${SRCDIR} -std=c11 -O3 -DNDEBUG -D_GNU_SOURCE -D_XOPEN_SOURCE=600 -DGGML_USE_CPU_ONLY=1 -DGGML_STATIC=1
// #cgo CXXFLAGS: -I${SRCDIR} -std=c++11 -O3 -DNDEBUG -D_GNU_SOURCE -D_XOPEN_SOURCE=600 -DGGML_USE_CPU_ONLY=1 -DGGML_STATIC=1
// #cgo LDFLAGS: -lm -lstdc++ -fopenmp -static
// #include "whisper_static.c"
import "C"

// This file exists solely to include the C code directly in the Go build
func init() {}
