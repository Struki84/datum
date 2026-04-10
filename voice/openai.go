package voice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator"
)

// voiceMap maps Lokutor voice constants to OpenAI TTS voices.
var voiceMap = map[orchestrator.Voice]string{
	orchestrator.VoiceF1: "nova",
	orchestrator.VoiceF2: "shimmer",
	orchestrator.VoiceF3: "fable",
	orchestrator.VoiceF4: "nova",
	orchestrator.VoiceF5: "shimmer",
	orchestrator.VoiceM1: "onyx",
	orchestrator.VoiceM2: "echo",
	orchestrator.VoiceM3: "alloy",
	orchestrator.VoiceM4: "onyx",
	orchestrator.VoiceM5: "echo",
}

type OpenAITTS struct {
	apiKey string
	model  string
	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewOpenAITTS(apiKey string, model string) *OpenAITTS {
	if model == "" {
		model = "tts-1"
	}
	return &OpenAITTS{
		apiKey: apiKey,
		model:  model,
	}
}

func (t *OpenAITTS) Name() string {
	return "openai_tts"
}

func (t *OpenAITTS) mapVoice(voice orchestrator.Voice) string {
	if v, ok := voiceMap[voice]; ok {
		return v
	}
	return "nova"
}

func (t *OpenAITTS) Synthesize(ctx context.Context, text string, voice orchestrator.Voice, lang orchestrator.Language) ([]byte, error) {
	var audio []byte
	err := t.StreamSynthesize(ctx, text, voice, lang, func(chunk []byte) error {
		audio = append(audio, chunk...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return audio, nil
}

func (t *OpenAITTS) StreamSynthesize(ctx context.Context, text string, voice orchestrator.Voice, lang orchestrator.Language, onChunk func([]byte) error) error {
	// Create a cancellable context so Abort() can interrupt in-flight requests
	ctx, cancel := context.WithCancel(ctx)
	t.mu.Lock()
	t.cancel = cancel
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.cancel = nil
		t.mu.Unlock()
		cancel()
	}()

	openaiVoice := t.mapVoice(voice)

	payload := fmt.Sprintf(
		`{"model":"%s","input":"%s","voice":"%s","response_format":"pcm"}`,
		t.model, escapeJSON(text), openaiVoice,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/speech",
		strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI TTS error (%d): %s", resp.StatusCode, string(body))
	}

	// Stream the response in chunks
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if chunkErr := onChunk(chunk); chunkErr != nil {
				return chunkErr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read TTS stream: %w", err)
		}
	}
}

func (t *OpenAITTS) Abort() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	return nil
}

func escapeJSON(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
