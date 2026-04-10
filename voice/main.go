package voice

// Lokutor Voice Assistant Demo
//
// A self-contained voice assistant using lokutor-orchestrator for voice orchestration.
// Captures mic input via miniaudio, transcribes with STT, generates response with LLM,
// and plays back synthesized speech via TTS.
//
// Prerequisites:
//   - go get github.com/lokutor-ai/lokutor-orchestrator
//   - go get github.com/gen2brain/malgo
//   - go get github.com/joho/godotenv
//
// Environment variables (or .env file):
//   OPENAI_API_KEY   - Used for Whisper STT and GPT LLM
//   LOKUTOR_API_KEY  - Used for Lokutor TTS
//
// Run:
//   go run main.go

//
// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"os/signal"
// 	"sync"
// 	"syscall"
// 	"time"
//
// 	"github.com/gen2brain/malgo"
// 	"github.com/joho/godotenv"
// 	"github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator"
// 	llmProvider "github.com/lokutor-ai/lokutor-orchestrator/pkg/providers/llm"
// 	sttProvider "github.com/lokutor-ai/lokutor-orchestrator/pkg/providers/stt"
// )
//
// const (
// 	sampleRate = 24000
// 	channels   = 1
// )
//
// func main() {
// 	// ── Load environment ──────────────────────────────────
// 	if err := godotenv.Load(); err != nil {
// 		log.Println("No .env file found, using system environment variables")
// 	}
//
// 	openaiKey := os.Getenv("OPENAI_API_KEY")
//
// 	if openaiKey == "" {
// 		log.Fatal("OPENAI_API_KEY is required")
// 	}
//
// 	// ── Initialize providers ─────────────────────────────
// 	stt := sttProvider.NewOpenAISTT(openaiKey, "whisper-1")
// 	stt.SetSampleRate(sampleRate)
//
// 	llm := llmProvider.NewOpenAILLM(openaiKey, "gpt-5")
// 	tts := NewOpenAITTS(openaiKey, "tts-1")
//
// 	// ── Configure orchestrator ───────────────────────────
// 	config := orchestrator.DefaultConfig()
// 	config.Language = orchestrator.LanguageEn
// 	config.SilenceTimeout = 0
// 	config.FirstSpeaker = orchestrator.FirstSpeakerUser
//
// 	vad := orchestrator.NewImprovedRMSVAD(config.BargeInVADThreshold, 200*time.Millisecond, sampleRate)
// 	vad.SetMinConfirmed(2)
//
// 	orch := orchestrator.NewWithVAD(stt, llm, tts, vad, config)
//
// 	// ── Create session ───────────────────────────────────
// 	session := orch.NewSessionWithDefaults("demo_user")
//
// 	systemPrompt := `You are a helpful voice assistant. Keep your responses concise and
// conversational since they will be spoken aloud. Avoid markdown formatting, bullet points,
// code blocks, or special characters. Respond naturally as if having a spoken conversation.`
// 	orch.SetSystemPrompt(session, systemPrompt)
//
// 	// ── Start managed stream ─────────────────────────────
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
//
// 	stream := orch.NewManagedStream(ctx, session)
// 	stream.SetEchoSampleRates(sampleRate, sampleRate)
// 	defer stream.Close()
//
// 	// ── Initialize miniaudio ─────────────────────────────
// 	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
// 	if err != nil {
// 		log.Fatal("Failed to init audio context:", err)
// 	}
// 	defer mctx.Uninit()
//
// 	// ── Audio state ──────────────────────────────────────
// 	var (
// 		playbackMu    sync.Mutex
// 		playbackBytes []byte
// 	)
//
// 	// Buffered channels for non-blocking audio I/O
// 	inputChan := make(chan []byte, 512)
// 	go func() {
// 		for chunk := range inputChan {
// 			_ = stream.Write(chunk)
// 		}
// 	}()
//
// 	// ── Audio callback (duplex: mic in + speaker out) ────
// 	onSamples := func(pOutput, pInput []byte, frameCount uint32) {
// 		// Mic input → orchestrator
// 		if pInput != nil {
// 			buf := make([]byte, len(pInput))
// 			copy(buf, pInput)
// 			select {
// 			case inputChan <- buf:
// 			default:
// 			}
// 		}
//
// 		// Speaker output ← orchestrator
// 		if pOutput != nil {
// 			bytesToRead := int(frameCount) * 2
//
// 			playbackMu.Lock()
// 			if len(playbackBytes) == 0 {
// 				for i := range pOutput {
// 					pOutput[i] = 0
// 				}
// 			} else if len(playbackBytes) < bytesToRead {
// 				copy(pOutput, playbackBytes)
// 				for i := len(playbackBytes); i < bytesToRead; i++ {
// 					pOutput[i] = 0
// 				}
// 				playbackBytes = nil
// 			} else {
// 				copy(pOutput, playbackBytes[:bytesToRead])
// 				playbackBytes = playbackBytes[bytesToRead:]
// 			}
// 			playbackMu.Unlock()
//
// 			// // Copy for echo suppression (pOutput gets reused by miniaudio)
// 			// echoCopy := make([]byte, len(pOutput))
// 			// copy(echoCopy, pOutput)
// 			// stream.RecordPlayedOutput(echoCopy)
// 			// stream.NotifyAudioPlayed()
// 		}
// 	}
//
// 	// ── Configure duplex audio device ────────────────────
// 	deviceConfig := malgo.DefaultDeviceConfig(malgo.Duplex)
// 	deviceConfig.Capture.Format = malgo.FormatS16
// 	deviceConfig.Capture.Channels = channels
// 	deviceConfig.Playback.Format = malgo.FormatS16
// 	deviceConfig.Playback.Channels = channels
// 	deviceConfig.SampleRate = sampleRate
// 	deviceConfig.Alsa.NoMMap = 1
//
// 	device, err := malgo.InitDevice(mctx.Context, deviceConfig, malgo.DeviceCallbacks{
// 		Data: onSamples,
// 	})
// 	if err != nil {
// 		log.Fatal("Failed to init audio device:", err)
// 	}
// 	defer device.Uninit()
//
// 	if err := device.Start(); err != nil {
// 		log.Fatal("Failed to start audio device:", err)
// 	}
//
// 	// ── Handle orchestrator events ───────────────────────
// 	go func() {
// 		currentGeneration := 0
// 		for event := range stream.Events() {
// 			switch event.Type {
//
// 			case orchestrator.UserSpeaking:
// 				fmt.Println("\n🎤 Listening...")
//
// 			case orchestrator.UserStopped:
// 				fmt.Println("⏳ Processing...")
//
// 			case orchestrator.TranscriptFinal:
// 				fmt.Printf("📝 You: %s\n", event.Data.(string))
//
// 			case orchestrator.BotThinking:
// 				playbackMu.Lock()
// 				playbackBytes = nil
// 				currentGeneration = event.Generation
// 				playbackMu.Unlock()
// 				fmt.Println("🤔 Thinking...")
//
// 			case orchestrator.BotResponse:
// 				if resp, ok := event.Data.(string); ok {
// 					fmt.Printf("💬 AI: %s\n", resp)
// 				}
//
// 			case orchestrator.BotSpeaking:
// 				bd := stream.GetLatencyBreakdown()
// 				if bd.UserToPlay > 0 {
// 					fmt.Printf("⏱️  Latency: STT=%dms | LLM=%dms | TTS=%dms | Total=%dms\n",
// 						bd.STT, bd.LLM, bd.LLMToTTSFirstByte, bd.UserToPlay)
// 				}
// 				fmt.Println("🔊 Speaking...")
//
// 			case orchestrator.AudioChunk:
// 				if event.Generation < currentGeneration {
// 					continue // Skip audio from old generations (interrupted)
// 				}
// 				chunk := event.Data.([]byte)
// 				playbackMu.Lock()
// 				playbackBytes = append(playbackBytes, chunk...)
// 				playbackMu.Unlock()
//
// 			case orchestrator.Interrupted:
// 				playbackMu.Lock()
// 				playbackBytes = nil
// 				currentGeneration = event.Generation
// 				playbackMu.Unlock()
// 				fmt.Println("🛑 Interrupted")
//
// 			case orchestrator.ErrorEvent:
// 				fmt.Printf("❌ Error: %v\n", event.Data)
// 			}
// 		}
// 	}()
//
// 	// ── Print banner and wait for shutdown ────────────────
// 	fmt.Println("═══════════════════════════════════════")
// 	fmt.Println("  Lokutor Voice Assistant Demo")
// 	fmt.Println("  Just start speaking!")
// 	fmt.Println("  Press Ctrl+C to quit")
// 	fmt.Println("═══════════════════════════════════════")
//
// 	sig := make(chan os.Signal, 1)
// 	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
// 	<-sig
//
// 	fmt.Println("\n👋 Shutting down...")
// 	_ = device.Stop()
// 	stream.Close()
// }

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/joho/godotenv"
	"github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator"
	llmProvider "github.com/lokutor-ai/lokutor-orchestrator/pkg/providers/llm"
	sttProvider "github.com/lokutor-ai/lokutor-orchestrator/pkg/providers/stt"
)

const (
	sampleRate = 24000
	channels   = 1

	// Minimum PCM bytes before we start playback.
	// Prevents tiny initial TTS chunks from hitting the speaker as noise.
	// 2400 bytes = 50ms of 24kHz mono 16-bit audio — enough to avoid pops.
	minPlaybackBuffer = 9600
)

type State int

const (
	Listening State = iota
	Processing
	Speaking
)

func (s State) String() string {
	switch s {
	case Listening:
		return "Listening"
	case Processing:
		return "Processing"
	case Speaking:
		return "Speaking"
	default:
		return "Unknown"
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	// ── Providers ────────────────────────────────────────
	stt := sttProvider.NewOpenAISTT(openaiKey, "whisper-1")
	stt.SetSampleRate(sampleRate)
	llm := llmProvider.NewOpenAILLM(openaiKey, "gpt-4.1")
	tts := NewOpenAITTS(openaiKey, "tts-1")

	// ── Config ───────────────────────────────────────────
	config := orchestrator.DefaultConfig()
	config.Language = orchestrator.LanguageEn
	config.SilenceTimeout = 0
	config.FirstSpeaker = orchestrator.FirstSpeakerUser

	// Higher threshold = less sensitive to mouth noises, sighs, clicks
	config.BargeInVADThreshold = 0.015

	// Length of pause needed to detect user stoped talking
	config.BargeInVADTrailWindow = 3000 * time.Millisecond

	vad := orchestrator.NewImprovedRMSVAD(
		config.BargeInVADThreshold,
		200*time.Millisecond,
		sampleRate,
	)
	// Require more consecutive frames of energy to trigger speech
	vad.SetMinConfirmed(4)

	orch := orchestrator.NewWithVAD(stt, llm, tts, vad, config)

	// ── Session ──────────────────────────────────────────
	session := orch.NewSessionWithDefaults("demo_user")
	orch.SetSystemPrompt(session, `You are a helpful voice assistant. 
Keep responses concise and conversational. No markdown, no bullet points, 
no code blocks. Speak naturally.`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := orch.NewManagedStream(ctx, session)
	stream.SetEchoSampleRates(sampleRate, sampleRate)
	defer stream.Close()

	// ── Shared audio state ───────────────────────────────
	var (
		mu             sync.Mutex
		playbackBytes  []byte
		state          = Listening
		playbackReady  bool          // flips true once buffer reaches minPlaybackBuffer
		watchdogCancel chan struct{} // signals watchdog to stop
	)

	// Mic → orchestrator (only when Listening)
	inputChan := make(chan []byte, 512)
	go func() {
		for chunk := range inputChan {
			mu.Lock()
			s := state
			mu.Unlock()
			if s == Listening {
				_ = stream.Write(chunk)
			}
		}
	}()

	// ── malgo callback ───────────────────────────────────
	onSamples := func(pOutput, pInput []byte, frameCount uint32) {
		// Mic capture
		if pInput != nil {
			buf := make([]byte, len(pInput))
			copy(buf, pInput)
			select {
			case inputChan <- buf:
			default:
			}
		}

		// Playback
		if pOutput != nil {
			bytesToRead := int(frameCount) * 2
			mu.Lock()

			// Check if we've buffered enough to start playing
			if state == Speaking && !playbackReady && len(playbackBytes) >= minPlaybackBuffer {
				playbackReady = true
			}

			if !playbackReady || len(playbackBytes) == 0 {
				// Output silence
				for i := range pOutput {
					pOutput[i] = 0
				}
			} else if len(playbackBytes) < bytesToRead {
				copy(pOutput, playbackBytes)
				for i := len(playbackBytes); i < bytesToRead; i++ {
					pOutput[i] = 0
				}
				playbackBytes = nil
			} else {
				copy(pOutput, playbackBytes[:bytesToRead])
				playbackBytes = playbackBytes[bytesToRead:]
			}

			mu.Unlock()
		}
	}

	// ── Audio device ─────────────────────────────────────
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer mctx.Uninit()

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
		log.Fatal(err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		log.Fatal(err)
	}

	// ── Event loop ───────────────────────────────────────
	go func() {
		for event := range stream.Events() {
			switch event.Type {

			case orchestrator.UserSpeaking:
				fmt.Println("\n🎤 Listening...")

			case orchestrator.UserStopped:
				mu.Lock()
				state = Processing
				// Create a new cancel channel for this watchdog
				cancel := make(chan struct{})
				watchdogCancel = cancel
				mu.Unlock()
				fmt.Println("⏳ Processing...")

				go func() {
					select {
					case <-time.After(5 * time.Second):
						// Timed out — check if still stuck
						mu.Lock()
						stuck := state == Processing
						mu.Unlock()

						if stuck {
							fmt.Println("🔄 Didn't catch that, prompting retry...")

							retryAudio, err := tts.Synthesize(
								ctx,
								"Sorry, I didn't catch that. Can you say that again?",
								orchestrator.VoiceF1,
								orchestrator.LanguageEn,
							)
							if err != nil {
								fmt.Printf("❌ TTS retry error: %v\n", err)
								mu.Lock()
								state = Listening
								mu.Unlock()
								fmt.Println("\n🎤 Ready, speak now...")
								return
							}

							mu.Lock()
							playbackBytes = retryAudio
							playbackReady = false
							state = Speaking
							mu.Unlock()

							for {
								time.Sleep(50 * time.Millisecond)
								mu.Lock()
								started := playbackReady
								mu.Unlock()
								if started {
									break
								}
							}

							for {
								time.Sleep(50 * time.Millisecond)
								mu.Lock()
								empty := len(playbackBytes) == 0
								mu.Unlock()
								if empty {
									time.Sleep(300 * time.Millisecond)
									mu.Lock()
									state = Listening
									playbackReady = false
									mu.Unlock()
									fmt.Println("\n🎤 Ready, speak now...")
									return
								}
							}
						}

					case <-cancel:
						// Pipeline proceeded normally, watchdog not needed
						return
					}
				}()

			case orchestrator.BotThinking:
				mu.Lock()
				state = Processing
				playbackBytes = nil
				playbackReady = false
				// Kill the watchdog — pipeline is running
				if watchdogCancel != nil {
					close(watchdogCancel)
					watchdogCancel = nil
				}
				mu.Unlock()
				fmt.Println("🤔 Thinking...")
			case orchestrator.TranscriptFinal:
				fmt.Printf("📝 You: %s\n", event.Data.(string))

			case orchestrator.BotResponse:
				if resp, ok := event.Data.(string); ok {
					fmt.Printf("💬 AI: %s\n", resp)
				}

			case orchestrator.BotSpeaking:
				mu.Lock()
				state = Speaking
				mu.Unlock()
				fmt.Println("🔊 Speaking...")

				// Monitor playback completion
				go func() {
					// Wait for playback to actually start
					for {
						time.Sleep(50 * time.Millisecond)
						mu.Lock()
						started := playbackReady
						mu.Unlock()
						if started {
							break
						}
					}

					// Wait for buffer to drain (playback finished)
					for {
						time.Sleep(50 * time.Millisecond)
						mu.Lock()
						empty := len(playbackBytes) == 0
						s := state
						mu.Unlock()

						if empty && s == Speaking {
							// Cooldown — let residual audio clear
							time.Sleep(300 * time.Millisecond)
							mu.Lock()
							if state == Speaking {
								state = Listening
								playbackReady = false
							}
							mu.Unlock()
							fmt.Println("\n🎤 Ready, speak now...")
							return
						}
					}
				}()

			case orchestrator.AudioChunk:
				chunk := event.Data.([]byte)
				mu.Lock()
				playbackBytes = append(playbackBytes, chunk...)
				mu.Unlock()

			case orchestrator.Interrupted:
				// Ignore — turn-based

			case orchestrator.ErrorEvent:
				fmt.Printf("❌ Error: %v\n", event.Data)
				mu.Lock()
				state = Listening
				playbackBytes = nil
				playbackReady = false
				mu.Unlock()
			}
		}
	}()

	// ── Banner ───────────────────────────────────────────
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  Voice Assistant (Turn-Based)")
	fmt.Println("  Speak, then wait for response.")
	fmt.Println("  Press Ctrl+C to quit")
	fmt.Println("═══════════════════════════════════════")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\n👋 Shutting down...")
	_ = device.Stop()
	stream.Close()
}
