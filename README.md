# DatumTVA

A voice-enabled AI assistant with a RAG knowledge base. Speak a question, get a spoken answer — the assistant retrieves context from an embedded document library, reasons over it using a ReAct agent loop, and responds via text-to-speech.

---

## Architecture

```
  Microphone
      │
      ▼
  VAD (Lokutor)         ← detects speech / silence
      │
      ▼
  STT (OpenAI Whisper)  ← speech → transcript
      │
      ▼
  ReAct Agent (GoLangGraph)
      │  ├── Pinecone Search tool   ← semantic retrieval
      │  ├── List Files tool
      │  ├── Read File tool
      │  ├── Web Search tool        ← Google (SERP) / DuckDuckGo
      │  └── Web Scraper tool       ← fetches and reads web pages
      │
      ▼
  LLM (OpenAI GPT-4.1)  ← reasoning + response generation
      │
      ▼
  TTS (OpenAI TTS)      ← response → speech
      │
      ▼
  Speaker
```

| Component | Role |
|---|---|
| [Lokutor Orchestrator](https://github.com/lokutor-ai/lokutor-orchestrator) | Voice I/O pipeline — VAD, STT routing, TTS routing, turn management |
| [GoLangGraph](https://github.com/Struki84/GoLangGraph) | ReAct agent graph (agent → tool loop → response) |
| [langchaingo](https://github.com/tmc/langchaingo) | LLM interface, tool definitions |
| OpenAI Whisper | Speech-to-text transcription |
| OpenAI GPT-4.1 | Language model for reasoning and responses |
| OpenAI TTS | Text-to-speech output |
| Pinecone | Vector store for document retrieval |
| Google SERP / DuckDuckGo | Web search — Google is primary, DuckDuckGo is the fallback |
| Web Scraper | Fetches and reads web page content as tool input |
| OpenRouter | Alternative LLM routing (via `OPENROUTER_API_KEY`) |
| [clipt](https://github.com/Struki84/clipt) | BubbleTea-based TUI with chat and voice modes |
| malgo (miniaudio) | Audio I/O — duplex mic/speaker at 24kHz |
| SQLite | Local chat history database — created automatically on first run in the project root |

The agent runs in **turn-based mode**: the mic is active while the user speaks, muted during processing and playback. A watchdog timer handles silence/noise with a fallback spoken response.

---

## Configuration

Create a file at `./config/my.config.json` with the following structure:

```json
{
  "OPENAI_API_KEY": "",
  "OPENROUTER_API_KEY": "",
  "SERP_API_KEY": "",
  "PineconeHost": "",
  "PineconeAPIKey": "",
  "PineconeNamespace": ""
}
```

Fill in your API keys before running. The app will not start without a valid config file at this path.

The document knowledge base is pre-loaded — files are already embedded and indexed in Pinecone. No additional ingestion step is required to use the app.

A local SQLite database for chat history is created automatically in the project root on first run. No setup needed.

---

## Quick Start (Pre-built Binaries)

Binaries for macOS and Linux are included in the repository root. No Go installation or build step needed.

**macOS:**
```bash
./datum-darwin
```

**Linux:**
```bash
./datum-linux
```

Make the binary executable first if needed:
```bash
chmod +x ./datum-darwin   # or datum-linux
```

> Make sure `./config/my.config.json` exists and is populated before running.

---

## Running from Source

### Prerequisites

- Go 1.21 or later
- A C compiler (CGo is required for audio — `gcc` on Linux, Xcode command line tools on macOS)
- No other system libraries needed — miniaudio links only against `libdl` on Linux (present on all distros) and uses CoreAudio on macOS natively

Install Go: https://go.dev/dl/

On macOS, install Xcode command line tools if not already present:
```bash
xcode-select --install
```

### Run directly

```bash
go run .
```

Dependencies will be fetched automatically on first run via Go modules.

### Build from source

```bash
# macOS
go build -o datum-darwin .

# Linux
go build -o datum-linux .
```

Cross-compiling between platforms requires setting `GOOS` and `CGO_ENABLED`:
```bash
# Build Linux binary from macOS (requires a cross C compiler)
GOOS=linux CGO_ENABLED=1 go build -o datum-linux .
```

For simplicity, it's recommended to build each binary natively on its target platform.

---

## Using the App

The TUI has two modes, switchable via **Tab**:

- **Chat mode** — type a message and press Enter to interact with the agent via text
- **Voice mode** — speak naturally; the VU meter in the bottom bar shows mic/playback levels. Wait for the 🎤 Listening... prompt before speaking

Press `Ctrl+C` to exit.
