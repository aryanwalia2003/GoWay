```mermaid 
sequenceDiagram
    autonumber
    actor Client
    participant Router as HTTP Router<br/>(cmd/serve.go)
    participant LogMW as Logging MW<br/>(middleware/logging.go)
    participant AuthMW as Auth MW<br/>(middleware/auth.go)
    participant ConcMW as Concurrency MW<br/>(middleware/concurrency.go)
    participant Handler as RenderHandler<br/>(handler/render_handler.go)
    participant Asm as FolioAssembler<br/>(assembler/folio_assembler.go)
    participant BC as Barcode Engine<br/>(barcode/png_helper.go)

    Client->>Router: POST /render/forward-shipping-label
    
    Router->>LogMW: Route request
    activate LogMW
    Note over LogMW: 1. Timer start<br/>2. Generate trace_id & inject in Context
    
    LogMW->>AuthMW: Next()
    activate AuthMW
    Note over AuthMW: Check X-API-Key Header
    
    alt Invalid API Key
        AuthMW-->>Client: 401 Unauthorized (Reject)
    else Valid API Key
        AuthMW->>ConcMW: Next()
        activate ConcMW
        Note over ConcMW: Acquire Slot: c.sem <- struct{}{}<br/>(Waits if concurrency limit reached)
        
        ConcMW->>Handler: ServeHTTP
        activate Handler
        Note over Handler: Decode JSON Request Body
        
        alt Missing template_id or Bad JSON
            Handler-->>Client: 400 Bad Request
        else Valid Request
            Handler->>Asm: Assemble(ctx, templateID, data)
            activate Asm
            Note over Asm: 1. json.Unmarshal payload to map<br/>2. Inject pre-loaded cachedLogoURI
            
            opt Barcode(s) Missing in Payload?
                Asm->>BC: RenderBarcodePNG(zippeeAwb)
                activate BC
                Note over BC: Encode Code128 -> NRGBA -> Compress
                BC-->>Asm: Return []byte (PNG)
                deactivate BC
                Note over Asm: Convert PNG to Base64 String
                
                Asm->>BC: RenderBarcodePNG(referenceCode)
                activate BC
                BC-->>Asm: Return []byte (PNG)
                deactivate BC
                Note over Asm: Convert PNG to Base64 String
            end
            
            Note over Asm: getTemplate(templateID)<br/>(Check Memory Cache OR Read HTML from disk)
            Note over Asm: folio.AddHTMLTemplate(...)<br/>Inject map data into HTML
            Note over Asm: doc.ToBytes()<br/>Convert HTML to PDF
            
            Asm-->>Handler: Return PDF []byte (or error)
            deactivate Asm
            
            Note over Handler: writePDFResponse()<br/>Content-Type: application/pdf
            Handler-->>ConcMW: HTTP 200 OK + PDF Bytes
        end
        deactivate Handler
        
        Note over ConcMW: defer: Release Slot (<-c.sem)
        ConcMW-->>AuthMW: Return
        deactivate ConcMW
    end
    
    AuthMW-->>LogMW: Return
    deactivate AuthMW
    
    Note over LogMW: Calculate time.Since(start)<br/>logger.LogRequest(status, duration, trace_id)
    LogMW-->>Router: Request chain completed
    deactivate LogMW
    
    Router-->>Client: Receive Final PDF Response```
