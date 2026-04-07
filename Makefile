APP     := awb-gen
GO      := go
FIXTURE := testdata/bench_5000.json
OUTPDF  := /tmp/awb-bench-out.pdf
PORT     ?= 8080
API_KEYS ?= my-secret-key

.PHONY: all build race serve test bench bench-mem profile-cpu profile-mem \
        fixture clean fmt vet lint tidy

# ── Default ──────────────────────────────────────────────────────────────────
all: tidy fmt vet build test

# ── Build ────────────────────────────────────────────────────────────────────
build:
	$(GO) build -v -trimpath -o $(APP) main.go

serve: build
	API_KEYS=$(API_KEYS) ./$(APP) serve --port $(PORT)

# Build with race detector enabled — use for development only; ~5-10× slower.
race:
	$(GO) build -race -v -o $(APP) main.go

# ── Test ─────────────────────────────────────────────────────────────────────
# Always run with -race. The race detector costs almost nothing in test context
# and catches SPSC channel bugs that would be invisible otherwise.
test:
	$(GO) test -race -v -count=1 -timeout=120s ./...

# Run tests with coverage report.
cover:
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

# ── Benchmarks ───────────────────────────────────────────────────────────────
# Run all benchmarks, 3 repetitions, with allocation tracking.
bench: build fixture
	$(GO) test -run='^$$' -bench=. -benchmem -count=3 -timeout=300s ./...

# Benchmark the full end-to-end pipeline at 5 000 labels and time wall clock.
bench-e2e: build fixture
	@echo "=== End-to-end benchmark: 5 000 labels ==="
	time ./$(APP) --input $(FIXTURE) --output $(OUTPDF)
	@wc -c < $(OUTPDF) | awk '{printf "Output PDF size: %.1f KB\n", $$1/1024}'

# ── Memory benchmark ─────────────────────────────────────────────────────────
bench-mem: build fixture
	@echo "=== Peak RSS benchmark via /usr/bin/time ==="
	/usr/bin/time -v ./$(APP) --input $(FIXTURE) --output $(OUTPDF) 2>&1 \
	  | grep -E "Maximum resident|Elapsed"

# ── Profiling ────────────────────────────────────────────────────────────────
# Run the binary with pprof server, then immediately capture a 30-second CPU
# profile. Requires the binary to be running with --pprof=localhost:6060.
profile-cpu: build fixture
	@echo "Starting awb-gen with pprof on localhost:6060 ..."
	./$(APP) --input $(FIXTURE) --output $(OUTPDF) --pprof localhost:6060 &
	@sleep 1
	mkdir -p pprof
	$(GO) tool pprof -pdf http://localhost:6060/debug/pprof/profile?seconds=20 \
	  > pprof/cpu.pdf
	@echo "CPU profile written to pprof/cpu.pdf"

profile-mem: build fixture
	@echo "Capturing heap profile ..."
	./$(APP) --input $(FIXTURE) --output $(OUTPDF) --pprof localhost:6060 &
	@sleep 2
	mkdir -p pprof
	$(GO) tool pprof -pdf http://localhost:6060/debug/pprof/heap \
	  > pprof/heap.pdf
	@echo "Heap profile written to pprof/heap.pdf"

# ── Fixture ──────────────────────────────────────────────────────────────────
fixture: $(FIXTURE)

$(FIXTURE):
	@mkdir -p testdata
	@echo "Generating 5 000-label benchmark fixture ..."
	bash scripts/gen_fixture.sh 5000 > $(FIXTURE)
	@echo "Fixture written to $(FIXTURE)"

# ── Code quality ─────────────────────────────────────────────────────────────
fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

# golangci-lint must be installed: https://golangci-lint.run/usage/install/
lint:
	golangci-lint run ./...

tidy:
	$(GO) mod tidy

# ── Clean ────────────────────────────────────────────────────────────────────
clean:
	rm -f $(APP)
	rm -f output.pdf $(OUTPDF)
	rm -f coverage.out coverage.html
	rm -rf pprof/
	rm -rf testdata/