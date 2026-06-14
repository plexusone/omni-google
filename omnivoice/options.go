package omnivoice

import "time"

const (
	// DefaultModel is the default Gemini Live model.
	DefaultModel = "gemini-2.0-flash-live"

	// DefaultVoice is the default voice.
	DefaultVoice = "Puck"

	// DefaultInputSampleRate is the input sample rate (16kHz).
	DefaultInputSampleRate = 16000

	// DefaultOutputSampleRate is the output sample rate (24kHz).
	DefaultOutputSampleRate = 24000

	// DefaultTemperature is the default temperature.
	DefaultTemperature = 1.0

	// DefaultConnectTimeout is the default connection timeout.
	DefaultConnectTimeout = 30 * time.Second
)

// Available voices for Gemini Live API.
const (
	VoicePuck   = "Puck"
	VoiceCharon = "Charon"
	VoiceKore   = "Kore"
	VoiceFenrir = "Fenrir"
	VoiceAoede  = "Aoede"
)

// Audio formats.
const (
	AudioMimeTypePCM16k = "audio/pcm;rate=16000"
	AudioMimeTypePCM24k = "audio/pcm;rate=24000"
)

// Config holds client configuration.
type Config struct {
	// APIKey is the Google API key.
	APIKey string

	// Model is the Gemini model to use.
	Model string

	// Voice is the voice for audio output.
	Voice string

	// Instructions is the system prompt.
	Instructions string

	// ResponseModalities specifies output types (["AUDIO", "TEXT"]).
	ResponseModalities []string

	// Tools are functions the model can call.
	Tools []ToolDef

	// Temperature controls randomness (0-2).
	Temperature float64

	// TopP controls nucleus sampling.
	TopP float64

	// TopK controls top-k sampling.
	TopK int

	// MaxOutputTokens limits response length.
	MaxOutputTokens int

	// ConnectTimeout is the timeout for connecting.
	ConnectTimeout time.Duration

	// EnableGoogleSearch enables grounding with Google Search.
	EnableGoogleSearch bool

	// EnableCodeExecution enables code execution.
	EnableCodeExecution bool
}

// Option configures the client.
type Option func(*Config)

// WithModel sets the Gemini model.
func WithModel(model string) Option {
	return func(c *Config) {
		c.Model = model
	}
}

// WithVoice sets the voice for audio output.
func WithVoice(voice string) Option {
	return func(c *Config) {
		c.Voice = voice
	}
}

// WithInstructions sets the system prompt.
func WithInstructions(instructions string) Option {
	return func(c *Config) {
		c.Instructions = instructions
	}
}

// WithResponseModalities sets the response modalities.
func WithResponseModalities(modalities ...string) Option {
	return func(c *Config) {
		c.ResponseModalities = modalities
	}
}

// WithTools sets the tools the model can call.
func WithTools(tools ...ToolDef) Option {
	return func(c *Config) {
		c.Tools = tools
	}
}

// WithFunctions adds function declarations as tools.
func WithFunctions(functions ...FunctionDeclaration) Option {
	return func(c *Config) {
		c.Tools = append(c.Tools, ToolDef{
			FunctionDeclarations: functions,
		})
	}
}

// WithTemperature sets the temperature (0-2).
func WithTemperature(temp float64) Option {
	return func(c *Config) {
		c.Temperature = temp
	}
}

// WithTopP sets the top-p value.
func WithTopP(topP float64) Option {
	return func(c *Config) {
		c.TopP = topP
	}
}

// WithTopK sets the top-k value.
func WithTopK(topK int) Option {
	return func(c *Config) {
		c.TopK = topK
	}
}

// WithMaxOutputTokens sets the maximum output tokens.
func WithMaxOutputTokens(max int) Option {
	return func(c *Config) {
		c.MaxOutputTokens = max
	}
}

// WithConnectTimeout sets the connection timeout.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = timeout
	}
}

// WithGoogleSearch enables grounding with Google Search.
func WithGoogleSearch() Option {
	return func(c *Config) {
		c.EnableGoogleSearch = true
	}
}

// WithCodeExecution enables code execution.
func WithCodeExecution() Option {
	return func(c *Config) {
		c.EnableCodeExecution = true
	}
}

// applyDefaults applies default values to the config.
func applyDefaults(c *Config) {
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.Voice == "" {
		c.Voice = DefaultVoice
	}
	if len(c.ResponseModalities) == 0 {
		c.ResponseModalities = []string{"AUDIO", "TEXT"}
	}
	if c.Temperature == 0 {
		c.Temperature = DefaultTemperature
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = DefaultConnectTimeout
	}
}
