package metrics

import (

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/apache/yunikorn-core/pkg/locking"
	"github.com/apache/yunikorn-core/pkg/log"
)

// CustomMetrics to declare scheduler metrics
type CustomMetrics struct {
    decisionTimeDuration        prometheus.Gauge
    finalDecisionScore          prometheus.Gauge
    finalZeroSolutionRatio      prometheus.Gauge
    initialCandidateAvgScore    prometheus.Gauge
	customCPUUsage				*prometheus.GaugeVec
	customMemoryUsage			*prometheus.GaugeVec
	lock                  locking.RWMutex
}

// InitCustomMetrics to initialize scheduler metrics
func initCustomMetrics() *CustomMetrics {
	c := &CustomMetrics{
		lock: locking.RWMutex{},
	}

	// Decision time duration
	c.decisionTimeDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: CustomSubsystem,
			Name:      "decision_time_duration_seconds",
			Help:      "The time duration (in seconds) taken from the start to the final decision in the metaheuristic algorithm",
		})
	
	// Final decision score
	c.finalDecisionScore = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: CustomSubsystem,
			Name:      "final_decision_score",
			Help:      "The score of the final decision solution in the metaheuristic algorithm",
		})

	// Final zero solution ratio
	c.finalZeroSolutionRatio = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: CustomSubsystem,
			Name:      "final_zero_solution_ratio",
			Help:      "The ratio of final decision solutions that are all zeros in the metaheuristic algorithm",
		})

	// Initial candidate average score
	c.initialCandidateAvgScore = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: CustomSubsystem,
			Name:      "initial_candidate_avg_score",
			Help:      "The average score of the initial candidate solutions in the metaheuristic algorithm",
		})

	c.customCPUUsage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: Namespace,
            Subsystem: CustomSubsystem,
            Name:      "cpu_usage_percent",
            Help:      "Current CPU usage percentage for each node",
        },
        []string{"node_name"}, // 節點名稱作為標籤
    )

	c.customMemoryUsage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: Namespace,
            Subsystem: CustomSubsystem,
            Name:      "memory_usage_percent",
            Help:      "Current memory usage percentage for each node",
        },
        []string{"node_name"}, // 節點名稱作為標籤
    )

	// Register the metrics
	var metricsList = []prometheus.Collector{
		c.decisionTimeDuration,
		c.finalDecisionScore,
		c.finalZeroSolutionRatio,
		c.initialCandidateAvgScore,
		c.customCPUUsage,
		c.customMemoryUsage,
	}
	for _, metric := range metricsList {
		if err := prometheus.Register(metric); err != nil {
			log.Log(log.Custom).Warn("failed to register metrics collector", zap.Error(err))
		}
	}
	return c
}

// Reset all metrics that implement the Reset functionality.
// should only be used in tests
func (m *CustomMetrics) Reset() {
}

func (c *CustomMetrics) SetDecisionTimeDuration(value float64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.decisionTimeDuration.Set(value)
}

func (c *CustomMetrics) SetFinalDecisionScore(value float64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.finalDecisionScore.Set(value)
}

func (c *CustomMetrics) SetFinalZeroSolutionRatio(value float64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.finalZeroSolutionRatio.Set(value)
}

func (c *CustomMetrics) SetInitialCandidateAvgScore(value float64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.initialCandidateAvgScore.Set(value)
}

func (c *CustomMetrics) SetCustomCPUUsage(name string, value float64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.customCPUUsage.With(prometheus.Labels{"node_name": name}).Set(value)
}

func (c *CustomMetrics) SetCustomMemoryUsage(name string, value float64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.customMemoryUsage.With(prometheus.Labels{"node_name": name}).Set(value)

}