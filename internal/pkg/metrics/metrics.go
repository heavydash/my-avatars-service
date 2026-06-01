package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RED metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	HTTPErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "path", "status"},
	)

	// Business metrics
	AvatarUploadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatar_uploads_total",
			Help: "Total number of avatar uploads",
		},
		[]string{"status"},
	)

	AvatarUploadDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "avatar_upload_duration_seconds",
			Help:    "Time spent uploading avatar",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
		},
	)

	AvatarDeletesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatar_deletes_total",
			Help: "Total number of avatar deletes",
		},
		[]string{"status"},
	)

	StorageUsageBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "avatar_storage_bytes",
			Help: "Total storage used by avatars in bytes",
		},
		[]string{"user_id"},
	)

	StorageObjectsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "avatar_storage_objects_total",
			Help: "Total number of avatar objects in storage",
		},
	)
)
