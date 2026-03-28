# Agent Coding Standards: Atomic Go Implementation

To ensure maximum modularity and performance for the AWB Generator, all agents must adhere to the following "Atomic Go" rules.

## 1. Structural Constraints (SRT - Single Responsibility Principle)
- **File Limit:** No file may exceed 50 lines of code.
- **Single Entity Rule:** Each file must contain exactly one logical entity (one struct, one interface, or one function/method).
- **Package Layout:** Logic must be separated into small, focused packages (e.g., `barcode`, `layout`, `pdfengine`, `cli`).

## 2. Naming Conventions (Strict)
Files and entities must be named based on their role:
- **Structs:** `[name]_struct.go` (e.g., `awb_struct.go`)
- **Constructors:** `[name]_ctor.go` (e.g., `awb_ctor.go`)
- **Methods:** `[method_name]_method.go` (e.g., `render_method.go`) - *Note: If a struct has 5 methods, they go in 5 separate files.*
- **Interfaces:** `[name]_iface.go` (e.g., `generator_iface.go`)
- **Constants:** `[name]_const.go` (e.g., `dimensions_const.go`)
- **Variables/Errors:** `[name]_var.go` or `[name]_err.go`

## 3. Coding Style
- **Self-Documenting:** Use descriptive names like `calculateTotalLabelHeight` instead of `calcH`.
- **Dependency Injection:** Methods should receive interfaces, not concrete types, to allow for easy mocking/testing.
- **Embedded Fonts:** Use `//go:embed` to include all assets inside the binary.
- **Error Handling:** Every function must return an `error` as the last return value. No `panic` allowed.

