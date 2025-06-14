package stellar

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// USDC transaction metrics
	usdcTransactionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stellar_usdc_transactions_total",
			Help: "Total number of USDC transactions processed",
		},
		[]string{"status"},
	)

	usdcTransactionAmount = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stellar_usdc_transaction_amount",
			Help:    "Amount of USDC in transactions",
			Buckets: prometheus.ExponentialBuckets(1, 10, 8), // 1 to 10^8 USDC
		},
		[]string{"type"},
	)

	usdcTransactionLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stellar_usdc_transaction_latency_seconds",
			Help:    "Time taken to process USDC transactions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	usdcTransactionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stellar_usdc_transaction_errors_total",
			Help: "Total number of USDC transaction processing errors",
		},
		[]string{"error_type"},
	)
)

func init() {
	// Register metrics
	prometheus.MustRegister(
		usdcTransactionTotal,
		usdcTransactionAmount,
		usdcTransactionLatency,
		usdcTransactionErrors,
	)
}

// RecordUSDCTransaction records metrics for a USDC transaction
func RecordUSDCTransaction(status string, amount float64, operation string, duration float64) {
	usdcTransactionTotal.WithLabelValues(status).Inc()
	usdcTransactionAmount.WithLabelValues("transfer").Observe(amount)
	usdcTransactionLatency.WithLabelValues(operation).Observe(duration)
}

// RecordUSDCTransactionError records metrics for USDC transaction errors
func RecordUSDCTransactionError(errorType string) {
	usdcTransactionErrors.WithLabelValues(errorType).Inc()
}
