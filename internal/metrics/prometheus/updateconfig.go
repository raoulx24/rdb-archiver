package prometheus

import "slices"

// UpdateConfig hot‑reloads source directory and file names.
func (s *promServer) UpdateConfig(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isSameConfig(cfg, s.cfg) {
		s.logg.Debug("same config, returning", "function", "UpdateConfig")
		return
	}
	s.cfg = cfg
	s.logg.Info("config updated")
}

// NeedsRestart reports whether watcher must be restarted for config change.
func (s *promServer) NeedsRestart(newCfg Config) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	oldCfg := s.cfg
	needRestart := !s.isSameConfig(oldCfg, newCfg)
	s.logg.Debug("check for if restart is needed", "function", "NeedsRestart", "isSameConfig", !needRestart)
	return needRestart
}

func (s *promServer) NeedMetricsRebuild(newCfg Config) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	oldCfg := s.cfg
	needMetricsRebuild := !slices.Equal(oldCfg.HistogramBuckets, newCfg.HistogramBuckets)
	s.logg.Debug("check for if metrics rebuild is needed", "function", "NeedMetricsRebuild", "needMetricsRebuild", needMetricsRebuild)
	return needMetricsRebuild
}

// isSameConfig compares the configs
func (s *promServer) isSameConfig(oldCfg, newCfg Config) bool {
	return oldCfg.Enabled == newCfg.Enabled &&
		oldCfg.Port == newCfg.Port &&
		slices.Equal(oldCfg.HistogramBuckets, newCfg.HistogramBuckets)
}
