# Login performance result

- Users: 100
- Target rate: 10 requests/s
- Warmup: 10s
- Measured duration: 1m0s
- Request timeout: 5s

## Vegeta report

```text
Requests      [total, rate, throughput]         600, 10.02, 10.02
Duration      [total, attack, wait]             59.902s, 59.901s, 1.622ms
Latencies     [min, mean, 50, 90, 95, 99, max]  1.532ms, 1.867ms, 1.76ms, 2.166ms, 2.458ms, 3.878ms, 5.535ms
Bytes In      [total, mean]                     218400, 364.00
Bytes Out     [total, mean]                     23400, 39.00
Success       [ratio]                           100.00%
Status Codes  [code:count]                      200:600  
Error Set:
```

- Throughput shows completed requests per second.
- Success shows the percentage of responses accepted by Vegeta.
- Latencies report request delay distribution; P95 and P99 expose slow-tail behavior.

## Service metrics during measured load

Counter values below are calculated from the after-minus-before difference. Runtime and connection values show the boundary state.

- Successful HTTP login requests: 600
- Average server-side login duration: 1.056ms
- Successful database queries: 600
- Average database query duration: 759µs
- Database connection waits: 0, total wait 0s
- Database connections after load: open 2, in use 0, idle 2
- Process CPU consumed during load: 750ms
- Resident memory after load: 43.14 MiB
- Go allocated heap boundary: 6.31 MiB -> 8.78 MiB
- Goroutines boundary: 28 -> 28

### Metric meaning

- `pgo_http_*`: HTTP request volume, errors, and latency.
- `pgo_db_query_*`: database query volume and latency.
- `pgo_db_connections_*`: open, active, idle, and waiting database connections.
- `go_*`: Go runtime memory, garbage collection, and goroutine state.
- `process_*`: operating-system resource usage for the service process.
