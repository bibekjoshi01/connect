// Package observability exposes bounded operational counters without user labels.
package observability

import (
	"database/sql"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

type Metrics struct {
	Requests           atomic.Uint64
	RequestNanoseconds atomic.Uint64
	ReadinessChecks    atomic.Uint64
	ReadinessFailures  atomic.Uint64
}

func (m *Metrics) ObserveRequest(duration time.Duration) { m.RequestNanoseconds.Add(uint64(duration)) }

func (m *Metrics) Write(w io.Writer, stats sql.DBStats) {
	// All names are static and there are no request paths, account IDs or labels.
	_, _ = fmt.Fprintf(w, `# TYPE connect_http_requests_total counter
connect_http_requests_total %d
# TYPE connect_http_request_duration_seconds_total counter
connect_http_request_duration_seconds_total %.9f
# TYPE connect_readiness_checks_total counter
connect_readiness_checks_total %d
# TYPE connect_readiness_failures_total counter
connect_readiness_failures_total %d
# TYPE connect_db_open_connections gauge
connect_db_open_connections %d
# TYPE connect_db_in_use_connections gauge
connect_db_in_use_connections %d
# TYPE connect_db_idle_connections gauge
connect_db_idle_connections %d
# TYPE connect_db_wait_total counter
connect_db_wait_total %d
`, m.Requests.Load(), float64(m.RequestNanoseconds.Load())/float64(time.Second), m.ReadinessChecks.Load(), m.ReadinessFailures.Load(), stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)
}
