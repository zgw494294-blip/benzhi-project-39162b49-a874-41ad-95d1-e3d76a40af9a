package checks

type Config struct {
	MinimumSafetyFactor float64
	MinimumClearanceCm  float64
}

func DefaultConfig() Config {
	return Config{
		MinimumSafetyFactor: 1.50,
		MinimumClearanceCm:  30,
	}
}

type Engine struct {
	config Config
}

func New(config Config) *Engine {
	if config.MinimumSafetyFactor <= 0 {
		config.MinimumSafetyFactor = DefaultConfig().MinimumSafetyFactor
	}
	if config.MinimumClearanceCm <= 0 {
		config.MinimumClearanceCm = DefaultConfig().MinimumClearanceCm
	}
	return &Engine{config: config}
}
