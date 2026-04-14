package agents

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator"
	sttProvider "github.com/lokutor-ai/lokutor-orchestrator/pkg/providers/stt"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/struki84/datum/config"
	"github.com/struki84/datum/voice"
)

const (
	sampleRate        = 24000
	channels          = 1
	minPlaybackBuffer = 9600
)

type voiceState int

const (
	voiceListening voiceState = iota
	voiceProcessing
	voiceSpeaking
)

type VoiceAgent struct {
	agent *ChatAgent

	// Lokutor
	orch    *orchestrator.Orchestrator
	session *orchestrator.ConversationSession
	stream  *orchestrator.ManagedStream

	// Audio
	mctx   *malgo.AllocatedContext
	device *malgo.Device

	// TTS — stored on struct so eventLoop and watchdog share the same instance
	tts *voice.OpenAITTS

	// Playback state
	mu             sync.Mutex
	playbackBytes  []byte
	playbackReady  bool
	lastChunkAt    time.Time
	watchdogCancel chan struct{}
	state          voiceState

	// TUI bridge
	streamHandler func(ctx context.Context, msg schema.Msg) error
	LVLChannel    chan float64

	ctx    context.Context
	cancel context.CancelFunc
}

func NewVoiceAgent(chatAgent *ChatAgent) *VoiceAgent {
	cnf := config.New()

	stt := sttProvider.NewOpenAISTT(cnf.OpenAIAPIKey, "whisper-1")
	stt.SetSampleRate(sampleRate)

	tts := voice.NewOpenAITTS(cnf.OpenAIAPIKey, "tts-1")

	cfg := orchestrator.DefaultConfig()
	cfg.SampleRate = sampleRate
	cfg.Channels = channels
	cfg.Language = orchestrator.LanguageEn
	cfg.SilenceTimeout = 0
	cfg.FirstSpeaker = orchestrator.FirstSpeakerUser
	cfg.BargeInVADThreshold = 0.015
	cfg.BargeInVADTrailWindow = 3000 * time.Millisecond

	vad := orchestrator.NewImprovedRMSVAD(
		cfg.BargeInVADThreshold,
		200*time.Millisecond,
		sampleRate,
	)
	vad.SetMinConfirmed(4)

	// chatAgent satisfies orchestrator.LLMProvider via Complete()
	orch := orchestrator.New(stt, chatAgent, tts, vad, cfg, &orchestrator.NoOpLogger{})

	return &VoiceAgent{
		agent:      chatAgent,
		orch:       orch,
		tts:        tts,
		LVLChannel: make(chan float64, 8),
	}
}

// ── ChatProvider interface ────────────────────────────────────────────────────

func (v *VoiceAgent) Name() string {
	return "Datum Voice Agent"
}

func (v *VoiceAgent) Type() schema.ProviderType {
	return schema.Agent
}

func (v *VoiceAgent) Description() string {
	return "Voice mode — speak to interact"
}

func (v *VoiceAgent) Stream(ctx context.Context, callback func(ctx context.Context, msg schema.Msg) error) {
	v.streamHandler = callback
}

// Run is a no-op for voice — input comes from the mic, not the text box.
func (v *VoiceAgent) Run(ctx context.Context, input string, session schema.ChatSession) (string, error) {
	return "", nil
}

// ── Activation / deactivation (called on tab switch) ─────────────────────────

func (v *VoiceAgent) Activate(session schema.ChatSession) error {
	log.Println("VoiceAgent: Activating...")
	v.ctx, v.cancel = context.WithCancel(context.Background())

	v.agent.SetSession(session)
	v.agent.Stream(v.ctx, v.streamHandler)

	v.session = v.orch.NewSessionWithDefaults(session.ID)
	v.orch.SetSystemPrompt(v.session, primerMsg)

	v.stream = v.orch.NewManagedStream(v.ctx, v.session)
	v.stream.SetEchoSampleRates(sampleRate, sampleRate)

	if err := v.startAudio(); err != nil {
		return fmt.Errorf("voice: audio init failed: %w", err)
	}

	go v.eventLoop()

	v.emit(schema.Msg{
		Role:      schema.SysMsg,
		Content:   "🎤 Listening...",
		Timestamp: time.Now().Unix(),
	})

	return nil
}

func (v *VoiceAgent) Deactivate() {
	log.Println("VoiceAgent: Deactivating...")
	if v.device != nil {
		_ = v.device.Stop()
		v.device.Uninit()
		v.device = nil
	}

	if v.mctx != nil {
		v.mctx.Uninit()
		v.mctx = nil
	}

	if v.stream != nil {
		v.stream.Close()
		v.stream = nil
	}

	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}

	v.mu.Lock()
	v.playbackBytes = nil
	v.playbackReady = false
	v.state = voiceListening
	v.mu.Unlock()
}

// ── Audio setup ───────────────────────────────────────────────────────────────

func (v *VoiceAgent) startAudio() error {
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return err
	}
	v.mctx = mctx

	inputChan := make(chan []byte, 512)
	go func() {
		for chunk := range inputChan {
			v.mu.Lock()
			s := v.state
			v.mu.Unlock()
			if s == voiceListening {
				_ = v.stream.Write(chunk)
			}
		}
	}()

	onSamples := func(pOutput, pInput []byte, frameCount uint32) {
		if pInput != nil {
			buf := make([]byte, len(pInput))
			copy(buf, pInput)
			select {
			case inputChan <- buf:
			default:
			}

			// Publish mic level while listening.
			v.mu.Lock()
			s := v.state
			v.mu.Unlock()

			if s == voiceListening {
				level := RMSFromPCM(pInput)
				select {
				case v.LVLChannel <- level:
				default:
				}
			}
		}

		if pOutput != nil {
			bytesToRead := int(frameCount) * 2
			v.mu.Lock()

			if v.state == voiceSpeaking && !v.playbackReady && len(v.playbackBytes) >= minPlaybackBuffer {
				v.playbackReady = true
			}

			if !v.playbackReady || len(v.playbackBytes) == 0 {
				for i := range pOutput {
					pOutput[i] = 0
				}
			} else if len(v.playbackBytes) < bytesToRead {
				copy(pOutput, v.playbackBytes)
				for i := len(v.playbackBytes); i < bytesToRead; i++ {
					pOutput[i] = 0
				}
				v.playbackBytes = nil
			} else {
				copy(pOutput, v.playbackBytes[:bytesToRead])
				v.playbackBytes = v.playbackBytes[bytesToRead:]
			}

			// Publish playback level while speaking.
			speaking := v.state == voiceSpeaking && v.playbackReady
			var outputSnap []byte
			if speaking && len(pOutput) > 0 {
				outputSnap = make([]byte, len(pOutput))
				copy(outputSnap, pOutput)
			}

			v.mu.Unlock()

			if outputSnap != nil {
				select {
				case v.LVLChannel <- RMSFromPCM(outputSnap):
				default:
				}
			}
		}
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Duplex)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = channels
	deviceConfig.Playback.Format = malgo.FormatS16
	deviceConfig.Playback.Channels = channels
	deviceConfig.SampleRate = sampleRate
	deviceConfig.Alsa.NoMMap = 1

	device, err := malgo.InitDevice(mctx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onSamples,
	})
	if err != nil {
		return err
	}
	v.device = device

	return device.Start()
}

// ── Event loop ────────────────────────────────────────────────────────────────

func (v *VoiceAgent) eventLoop() {
	for event := range v.stream.Events() {
		log.Printf("eventLoop: received event type=%v", event.Type)
		switch event.Type {

		case orchestrator.UserSpeaking:

		case orchestrator.UserStopped:
			v.mu.Lock()
			v.state = voiceProcessing
			cancel := make(chan struct{})
			v.watchdogCancel = cancel
			v.mu.Unlock()

			go v.watchdog(cancel)

		case orchestrator.TranscriptFinal:
			text, _ := event.Data.(string)
			v.emit(schema.Msg{
				Role:      schema.UserMsg,
				Content:   text,
				Timestamp: time.Now().Unix(),
			})

		case orchestrator.BotThinking:
			v.mu.Lock()
			v.state = voiceProcessing
			v.playbackBytes = nil
			v.playbackReady = false
			if v.watchdogCancel != nil {
				close(v.watchdogCancel)
				v.watchdogCancel = nil
			}
			v.mu.Unlock()

			v.emit(schema.Msg{
				Role:      schema.SysMsg,
				Content:   "🤔 Thinking...",
				Timestamp: time.Now().Unix(),
			})

		case orchestrator.BotResponse:

		case orchestrator.BotSpeaking:
			v.mu.Lock()
			v.state = voiceSpeaking
			v.lastChunkAt = time.Now()
			v.mu.Unlock()

			go v.monitorPlayback()

		case orchestrator.AudioChunk:
			chunk, _ := event.Data.([]byte)
			v.mu.Lock()
			v.playbackBytes = append(v.playbackBytes, chunk...)
			v.lastChunkAt = time.Now()
			v.mu.Unlock()

		case orchestrator.ErrorEvent:
			log.Printf("voice error: %v", event.Data)
			v.emit(schema.Msg{
				Role:      schema.ErrMsg,
				Content:   fmt.Sprintf("Voice error: %v", event.Data),
				Timestamp: time.Now().Unix(),
			})
			v.mu.Lock()
			v.state = voiceListening
			v.playbackBytes = nil
			v.playbackReady = false
			v.mu.Unlock()
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (v *VoiceAgent) emit(msg schema.Msg) {
	if v.streamHandler == nil {
		return
	}
	if err := v.streamHandler(v.ctx, msg); err != nil {
		log.Printf("voice: stream handler error: %v", err)
	}
}

func (v *VoiceAgent) watchdog(cancel chan struct{}) {
	select {
	case <-time.After(60 * time.Second):
		v.mu.Lock()
		stuck := v.state == voiceProcessing
		v.mu.Unlock()

		if !stuck {
			return
		}

		retryAudio, err := v.tts.Synthesize(
			v.ctx,
			"Sorry, I didn't catch that. Can you say that again?",
			orchestrator.VoiceF1,
			orchestrator.LanguageEn,
		)
		if err != nil {
			v.mu.Lock()
			v.state = voiceListening
			v.mu.Unlock()
			return
		}

		v.mu.Lock()
		v.playbackBytes = retryAudio
		v.playbackReady = false
		v.lastChunkAt = time.Now()
		v.state = voiceSpeaking
		v.mu.Unlock()

		v.monitorPlayback()

	case <-cancel:
		return
	}
}

func (v *VoiceAgent) monitorPlayback() {
	// Wait for playback to actually start (buffer threshold met)
	for {
		time.Sleep(50 * time.Millisecond)
		v.mu.Lock()
		started := v.playbackReady
		v.mu.Unlock()
		if started {
			break
		}
		log.Printf("monitorPlayback: waiting for playbackReady...")
	}

	// Wait until buffer is empty AND no new chunk has arrived for 300ms
	for {
		time.Sleep(50 * time.Millisecond)
		v.mu.Lock()
		empty := len(v.playbackBytes) == 0
		chunksDone := time.Since(v.lastChunkAt) > 300*time.Millisecond
		s := v.state
		bufLen := len(v.playbackBytes)
		chunkAge := time.Since(v.lastChunkAt)
		v.mu.Unlock()

		log.Printf("monitorPlayback: state=%v empty=%v chunksDone=%v bufLen=%d chunkAge=%v", s, empty, chunksDone, bufLen, chunkAge)

		if empty && chunksDone && s == voiceSpeaking {
			time.Sleep(200 * time.Millisecond)
			v.mu.Lock()
			if v.state == voiceSpeaking {
				v.state = voiceListening
				v.playbackReady = false
			}
			v.mu.Unlock()

			v.emit(schema.Msg{
				Role:      schema.SysMsg,
				Content:   "🎤 Listening...",
				Timestamp: time.Now().Unix(),
			})
			return
		}
	}
}
