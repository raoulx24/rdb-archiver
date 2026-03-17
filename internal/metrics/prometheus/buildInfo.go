package prometheus

import "github.com/prometheus/client_golang/prometheus"

type BuildInfoMetrics struct {
	buildInfo *prometheus.GaugeVec
}

func NewBuildInfoMetrics(reg prometheus.Registerer, version, commit, buildDate string) BuildInfoMetrics {
	m := BuildInfoMetrics{
		buildInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rdb_archiver_build_info",
				Help: "RDB Archiver build info",
			},
			[]string{"version", "commit", "build_date"},
		),
	}
	reg.MustRegister(m.buildInfo)

	// set the gauge to 1 with your labels
	m.buildInfo.WithLabelValues(version, commit, buildDate).Set(1)
	return m
}
