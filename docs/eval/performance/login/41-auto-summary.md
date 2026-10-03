# Login automatic load result

Built-in RPS ladder: `200, 400, 600, 800, 1000`.

| RPS | Requests | Throughput | Success | P50 | P95 | P99 | Status |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 200 | 12000 | 200.01 | 100.00% |  |  |  | passed |
| 400 | 24000 | 400.00 | 100.00% |  |  |  | passed |
| 600 | 35999 | 599.77 | 100.00% |  |  |  | passed |
| 800 | 48000 | 799.99 | 100.00% |  |  |  | passed |
| 1000 | 59999 | 733.09 | 79.44% |  |  |  | failed: automatic load stopped at 1000 RPS: success ratio 79.44% |

Each stage directory contains its load parameters, scenario data, raw Vegeta result, and text report.
