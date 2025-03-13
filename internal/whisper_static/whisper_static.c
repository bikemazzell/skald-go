// This file is a wrapper to include all necessary whisper source files directly
// This allows us to build a truly static binary without external dependencies

// Include the whisper implementation directly
#define WHISPER_IMPLEMENTATION
#include "whisper.h"

// Include the GGML implementation directly
#define GGML_IMPLEMENTATION
#include "ggml.h"

// Include the GGML allocation implementation directly
#define GGML_ALLOC_IMPLEMENTATION
#include "ggml-alloc.h"

// No need to include other files as they are included by the above headers
