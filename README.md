# jpact

`jpact` is a small, fast JSON compaction library with selectable drivers. It is
built for streaming workloads and provides both a clean top-level API and
exposed driver packages when you want explicit control.

## Why jpact

- **Streaming compaction**: process JSON without loading it all into memory.
- **Strict JSON validation**: rejects invalid JSON (e.g., trailing commas).
- **Selectable drivers**: internal compactor or a jsonv2-tokenizer-based driver.
- **Low allocation**: pooled, reusable buffers for steady-state workloads.

## Installation

```
go get pkt.systems/jpact
```

## Usage

### Default (internal) driver

```go
package main

import (
	"bytes"
	"strings"

	"pkt.systems/jpact"
)

func main() {
	input := `{"a": 1, "b": [true, false, null]}`
	var out bytes.Buffer

	c := jpact.New() // default internal driver
	if err := c.CompactWriter(&out, strings.NewReader(input), 0); err != nil {
		panic(err)
	}

	// out.String() == {"a":1,"b":[true,false,null]}
}
```

### Select the jsonv2 driver

```go
c := jpact.New(jpact.WithDriver(jpact.DriverJSONv2))
```

### Package helpers

```go
var out bytes.Buffer
_ = jpact.CompactWriter(&out, strings.NewReader(`{"a": 1}`), 0)
```

Note: package helpers always use the internal driver and do not modify any
package-level state.

### Driver packages (explicit)

```go
import "pkt.systems/jpact/compactor"
import "pkt.systems/jpact/jsonv2compactor"

_ = compactor.CompactWriter(w, r, 0)
_ = jsonv2compactor.CompactWriter(w, r, 0)
```

## API surface

- `jpact.New(opts ...Option) Compactor`
- `jpact.WithDriver(Driver)`
- `jpact.CompactWriter(w, r, maxBytes)`
- `jpact.CompactToBuffer(r, maxBytes)`

`maxBytes <= 0` disables the limit. When enabled, the compactor returns an error
once the input exceeds the limit.

## Performance

Benchmarks (Go 1.25.5, linux/amd64, 13th Gen Intel(R) Core(TM) i7-1355U) run with:

```
go test ./compactor -run '^$' -bench . -benchmem
```

Results (ns/op):

```
Small:
  encoding_json      302.2
  compactor          391.4
  jsonv2compactor    370.5

Medium:
  encoding_json      59,340
  compactor          46,300
  jsonv2compactor    36,695

Large:
  encoding_json      3,487,766
  compactor          709,888
  jsonv2compactor    874,995
```

Allocations in these benchmarks are **0 allocs/op** for all jpact drivers after
pooling and benchmark harness adjustments. Real-world allocation counts will
vary depending on your output writer and workload.

## Fuzzing

Fuzzers are wired for both drivers and can be run independently:

```
go test ./compactor -run Fuzz -fuzz=Fuzz -fuzztime=30s
go test ./jsonv2compactor -run Fuzz -fuzz=Fuzz -fuzztime=30s
```

These were last run on 2025-12-23 and completed successfully.

## Behavior notes

- Strict JSON validation (no trailing commas).
- Invalid UTF-8 within strings is allowed to match `encoding/json` behavior.
- Compactors are stateless and safe for concurrent use.

## License

See `LICENSE`.
