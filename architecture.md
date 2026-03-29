# GoWay: Deep Dive Architecture & Mechanics

This document provides a **painfully detailed, code-level breakdown** of the 6 major components that power the `awb-gen` high-performance PDF pipeline. 

Each component corresponds to a specific stage in the Single-Producer, Multi-Worker, Single-Consumer (SPSC) architecture.

---

## 1. The Ingestion Engine (The Producer)
**Location:** `internal/pipeline/produce_method.go`  
**Purpose:** Safely ingest a theoretically infinite stream of JSON objects without ever loading the entire array into RAM.

```mermaid
sequenceDiagram
    participant Main as Pipeline.Run()
    participant Prod as Pipeline.produce()
    participant JSON as goccy/go-json
    participant Chan as jobs (chan Job)

    Main->>Prod: go p.produce(ctx, stdin, jobs)
    activate Prod
    Prod->>JSON: NewDecoder(stdin)
    
    note over Prod, JSON: Read the opening bracket '['
    Prod->>JSON: dec.Token()
    
    loop dec.More() == true
        Prod->>JSON: dec.Decode(&record)
        
        alt Decode Error
            Prod-->>Prod: Log Warning, index++
        else Valid Decode
            Prod->>Prod: record.Validate()
            alt Invalid Record
                Prod-->>Prod: Log Warning, index++
            else Valid Record
                alt ctx.Done()
                    Prod-->>Main: Return (Halt)
                else
                    Prod->>Chan: Send Job{Index, Record}
                end
                Prod->>Prod: index++
            end
        end
    end
    
    Prod->>Chan: close(jobs)
    deactivate Prod
```

---

## 2. The Concurrency Orchestrator (Worker Pool)
**Location:** `internal/pipeline/run_method.go` & `work_method.go`  
**Purpose:** Manage exactly N worker goroutines (usually bound to physical CPU cores) to process the `jobs` channel concurrently. It explicitly handles graceful shutdown and channel closing.

```mermaid
graph TD
    subgraph Pipeline.Run
        Init["Init channels: jobs (100), results (100)"]
        Init --> StartProd["go produce()"]
        
        StartProd --> WG["wg := sync.WaitGroup{}<br/>wg.Add(workers)"]
        
        subgraph Worker Loop
            WG --> Spawn["go p.work(ctx, jobs, results)"]
        end
        
        Spawn --> Closer["go func() { wg.Wait(); close(results) }()"]
        Closer --> Merger["merger.MergeToFile(results)"]
    end

    subgraph Pipeline.work
        JobChan[/"jobs chan"/] --> Loop{"range jobs"}
        Loop --> |Next Job| ContextCheck{"ctx.Err() != nil?"}
        ContextCheck --> |Yes| Exit["Return"]
        ContextCheck --> |No| Render["gen.GenerateLabel(job.Record)"]
        Render --> ResultChan[/"results <- PageResult"/]
    end
```

---

## 3. The Label Generator (Rendering Engine)
**Location:** `internal/generator/generate_method.go` & `maroto_ctor.go`  
**Purpose:** Converts a single `AWB` struct into a standalone PDF byte slice. Following our latest optimizations, the configuration (`*entity.Config`) is cached, entirely eliminating font overhead per-label.

```mermaid
classDiagram
    class MarotoGenerator {
        -barcodeEncoder: barcode.Renderer
        -regularFont: byte array
        -boldFont: byte array
        -cfg: entity.Config
        +GenerateLabel(record awb.AWB) (byte array, error)
    }

    class Maroto {
        <<interface>>
        +Generate() document.Document
    }
```

```mermaid
sequenceDiagram
    participant Worker as p.work()
    participant Gen as MarotoGenerator
    participant Maroto engine
    participant Barcode as barcode.Renderer

    Worker->>Gen: GenerateLabel(record)
    activate Gen
    note over Gen: Reuse pre-compiled g.cfg
    Gen->>Maroto engine: maroto.New(g.cfg)
    
    Gen->>Maroto engine: addHeaderRows()
    Gen->>Maroto engine: addBodyRows()
    
    Gen->>Barcode: renderBarcodePNG(awb)
    Barcode-->>Gen: byte array (PNG)
    Gen->>Maroto engine: addBarcodeRows(pngBytes)
    
    Gen->>Maroto engine: m.Generate()
    Maroto engine-->>Gen: doc.Document
    
    note over Gen: Pre-allocate target buffer (5KB)
    Gen->>Gen: buf := make([]byte, 0, 5120)
    Gen->>Gen: append(buf, doc.GetBytes()...)
    
    Gen-->>Worker: byte array (Valid PDF), error
    deactivate Gen
```

---

## 4. The Barcode Engine
**Location:** `internal/barcode/render_method.go`  
**Purpose:** Converts alphanumeric AWB strings into highly readable Code128 PNG bitmaps that `maroto` can embed.

```mermaid
graph TD
    Input["Input: AWB String e.g. 'AWB123456'"] --> Code128["code128.Encode(awb)"]
    Code128 --> Rescale["barcode.Scale(bar, 250, 50)"]
    
    subgraph PNG Rasterization
        Rescale --> Buffer["bytes.NewBuffer(make(byte array, 0, 1024))"]
        Buffer --> Encode["png.Encode(buffer, scaled_barcode)"]
    end
    
    Encode --> Out["Return byte array (PNG image data)"]
```

---

## 5. The Windowed Merger (The O(1) Optimization)
**Location:** `internal/merger/merge_method.go`  
**Purpose:** This is the heart of the system's memory stability. Because Worker 8 might finish Job 12 before Worker 1 finishes Job 11, the results arrive **out of order**. The merger buffers them in a Min-Heap and flushes them.

```mermaid
stateDiagram-v2
    [*] --> WaitResults
    WaitResults --> PushHeap : Receive PageResult
    
    state "Min-Heap Processing" as HeapOpt {
        PushHeap --> CheckHead : heap.Push(res)
        CheckHead --> PopHead : heap 0 Index == ExpectedIndex
        PopHead --> AppendBuffer : heap.Pop()
        AppendBuffer --> CheckHead : ExpectedIndex++
        
        CheckHead --> WaitResults : heap 0 Index > ExpectedIndex
    }

    state "Chunk Flushing" as FlushOpt {
        AppendBuffer --> TriggerFlush : len(chunkBuffer) == 500
        TriggerFlush --> AppendToDisk : pdfcpu.MergeRaw APPEND
        AppendToDisk --> ClearMemory : chunkBuffer = make([]PageResult, 0)
        ClearMemory --> WaitResults : GC reclaims old page bytes
    }
```

---

## 6. The Integration Bridge (Django & CLI)
**Location:** `cmd/root.go` & `reverse_awb_generation.py`  
**Purpose:** How Python communicates with the compiled Go binary. Python uses `subprocess` to stream JSON.

```mermaid
sequenceDiagram
    participant Django as Celery Worker
    participant Subprocess as OS Subprocess PIPE
    participant GoBin as awb-gen
    participant Disk as /tmp/awb-xxx.pdf

    Django->>Django: Build list of AWB Dicts
    Django->>Django: json_data = json.dumps(list)
    
    Django->>Subprocess: Popen(['awb-gen', '--stdin', '--output', path])
    activate Subprocess
    
    Subprocess->>GoBin: Stream JSON over STDIN
    activate GoBin
    
    GoBin->>GoBin: Pipeline executes...
    GoBin->>Disk: Incremental chunk writes
    
    GoBin-->>Subprocess: Exit 0
    deactivate GoBin
    
    Subprocess-->>Django: Process Complete
    deactivate Subprocess
    
    Django->>Disk: Read final PDF
    Django->>Django: Upload to AWS S3 bucket
```
