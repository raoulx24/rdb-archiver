package snapshotwatcher

type Config struct {
	Path        string   `yaml:"path"`
	PrimaryName string   `yaml:"primaryName"`
	AuxNames    []string `yaml:"auxNames"`
	WatchMode   string   `yaml:"watchMode"`
}
