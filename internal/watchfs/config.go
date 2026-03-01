package watchfs

type Config struct {
	FSNotify        FSNotifyConfig `yaml:"fsnotify"`
	Pool            PoolConfig     `yaml:"pool"`
	StabilityWindow string         `yaml:"stabilityWindow"`
}

type FSNotifyConfig struct {
	DebounceWindow string `yaml:"debounceWindow"`
}

type PoolConfig struct {
	Interval string `yaml:"interval"`
}
