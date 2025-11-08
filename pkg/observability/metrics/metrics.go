package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

var (
	ingestionAccepted    atomic.Int64
	ingestionPublished   atomic.Int64
	ingestionFailed      atomic.Int64
	ingestionBacklog     atomic.Int64
	ingestionThroughput  atomic.Int64
	dlpRejectedToday     atomic.Int64
	dlpAcceptedToday     atomic.Int64
	tokenVaultSizeMetric atomic.Int64
)

func Init() {}

func ObservePipelineCounts(accepted, published, failed, backlog, throughput int) {
	ingestionAccepted.Store(int64(accepted))
	ingestionPublished.Store(int64(published))
	ingestionFailed.Store(int64(failed))
	ingestionBacklog.Store(int64(backlog))
	ingestionThroughput.Store(int64(throughput))
}

func ObserveDLPCounts(failed, accepted, tokenVault int) {
	dlpRejectedToday.Store(int64(failed))
	dlpAcceptedToday.Store(int64(accepted))
	tokenVaultSizeMetric.Store(int64(tokenVault))
}

func WritePrometheus(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP synaptica_pipeline_ingestion_accepted_total Number of ingestion requests accepted in the latest sampling window.\n")
	fmt.Fprintf(w, "# TYPE synaptica_pipeline_ingestion_accepted_total gauge\n")
	fmt.Fprintf(w, "synaptica_pipeline_ingestion_accepted_total %d\n", ingestionAccepted.Load())

	fmt.Fprintf(w, "# HELP synaptica_pipeline_ingestion_published_total Number of ingestion requests published in the latest sampling window.\n")
	fmt.Fprintf(w, "# TYPE synaptica_pipeline_ingestion_published_total gauge\n")
	fmt.Fprintf(w, "synaptica_pipeline_ingestion_published_total %d\n", ingestionPublished.Load())

	fmt.Fprintf(w, "# HELP synaptica_pipeline_ingestion_failed_total Number of ingestion requests failed in the latest sampling window.\n")
	fmt.Fprintf(w, "# TYPE synaptica_pipeline_ingestion_failed_total gauge\n")
	fmt.Fprintf(w, "synaptica_pipeline_ingestion_failed_total %d\n", ingestionFailed.Load())

	fmt.Fprintf(w, "# HELP synaptica_pipeline_ingestion_backlog_total Number of ingestion requests pending publication.\n")
	fmt.Fprintf(w, "# TYPE synaptica_pipeline_ingestion_backlog_total gauge\n")
	fmt.Fprintf(w, "synaptica_pipeline_ingestion_backlog_total %d\n", ingestionBacklog.Load())

	fmt.Fprintf(w, "# HELP synaptica_pipeline_ingestion_throughput_per_minute Number of ingestion requests processed per minute in the latest window.\n")
	fmt.Fprintf(w, "# TYPE synaptica_pipeline_ingestion_throughput_per_minute gauge\n")
	fmt.Fprintf(w, "synaptica_pipeline_ingestion_throughput_per_minute %d\n", ingestionThroughput.Load())

	fmt.Fprintf(w, "# HELP synaptica_privacy_dlp_rejected_today_total Number of payloads rejected by DLP today.\n")
	fmt.Fprintf(w, "# TYPE synaptica_privacy_dlp_rejected_today_total gauge\n")
	fmt.Fprintf(w, "synaptica_privacy_dlp_rejected_today_total %d\n", dlpRejectedToday.Load())

	fmt.Fprintf(w, "# HELP synaptica_privacy_dlp_accepted_today_total Number of payloads accepted by DLP today.\n")
	fmt.Fprintf(w, "# TYPE synaptica_privacy_dlp_accepted_today_total gauge\n")
	fmt.Fprintf(w, "synaptica_privacy_dlp_accepted_today_total %d\n", dlpAcceptedToday.Load())

	fmt.Fprintf(w, "# HELP synaptica_privacy_token_vault_size Number of active tokens in the de-identification vault.\n")
	fmt.Fprintf(w, "# TYPE synaptica_privacy_token_vault_size gauge\n")
	fmt.Fprintf(w, "synaptica_privacy_token_vault_size %d\n", tokenVaultSizeMetric.Load())
}
