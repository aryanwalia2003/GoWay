package assembler

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"awb-gen/internal/barcode"

	"github.com/tidwall/gjson"
)

type warmProcess struct {
	outDir string
	srcDir string
}

type LaTeXAssembler struct {
	tectonicBin string
	templateDir string
	warmPool    chan *warmProcess
	HardCap     time.Duration
	BarcodeRend barcode.Renderer
}

func NewLaTeXAssembler(bin, dir string, maxWorkers int) *LaTeXAssembler {
	absBin, _ := filepath.Abs(bin)
	absDir, _ := filepath.Abs(dir)
	a := &LaTeXAssembler{
		tectonicBin: absBin,
		templateDir: absDir,
		warmPool:    make(chan *warmProcess, maxWorkers),
		HardCap:     5 * time.Second,
		BarcodeRend: barcode.NewCode128Encoder(),
	}

	for i := 0; i < maxWorkers; i++ {
		a.warmPool <- a.spawnWarmProcess()
	}

	return a
}

func (a *LaTeXAssembler) spawnWarmProcess() *warmProcess {
	outDir, err := os.MkdirTemp("", "tectonic_out")
	if err != nil {
		outDir = "" // Fails loudly later
	}
	srcDir := filepath.Join(outDir, "src")
	os.MkdirAll(srcDir, 0755)

	return &warmProcess{
		outDir: outDir,
		srcDir: srcDir,
	}
}

func (a *LaTeXAssembler) Assemble(ctx context.Context, templateID string, payload []byte) ([]byte, error) {
	wp := <-a.warmPool
	go func() {
		a.warmPool <- a.spawnWarmProcess()
	}()

	defer os.RemoveAll(wp.outDir)

	ctx, cancel := context.WithTimeout(ctx, a.HardCap)
	defer cancel()

	return a.executeWarmProcess(ctx, wp, templateID, payload)
}

func (a *LaTeXAssembler) executeWarmProcess(ctx context.Context, wp *warmProcess, templateID string, load []byte) ([]byte, error) {
	// 1. Write Tectonic.toml
	tomlStr := `[doc]
name = "texput"
bundle = "https://relay.fullyjustified.net/default_bundle_v33.tar"
[[output]]
name = "texput"
type = "pdf"
`
	if err := os.WriteFile(filepath.Join(wp.outDir, "Tectonic.toml"), []byte(tomlStr), 0644); err != nil {
		return nil, fmt.Errorf("failed to write TOML: %v", err)
	}

	// 2. Read and write preamble to src/_preamble.tex
	preambleSrc := filepath.Join(a.templateDir, templateID+"_preamble.tex")
	preambleContent, err := os.ReadFile(preambleSrc)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read preamble: %v", err)
	}
	if len(preambleContent) == 0 {
		preambleContent = []byte("")
	}
	if err := os.WriteFile(filepath.Join(wp.srcDir, "_preamble.tex"), preambleContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to write _preamble.tex: %v", err)
	}

	// 3. Read body
	bodySrc := filepath.Join(a.templateDir, templateID+".tex")
	bodyContent, err := os.ReadFile(bodySrc)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %v", err)
	}

	// 4. Generate macros and barcodes
	macros := ""
	if len(load) > 0 {
		macros, _ = MapToMacros(load)
		if err := a.extractOrGenerateBarcodes(load, wp.srcDir); err != nil {
			return nil, fmt.Errorf("failed to extract or generate barcodes: %v", err)
		}
	}

	// 5. Combine macros + body into index.tex
	indexContent := macros + "\n" + string(bodyContent)
	if err := os.WriteFile(filepath.Join(wp.srcDir, "index.tex"), []byte(indexContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write index.tex: %v", err)
	}

	// 6. Execute Tectonic V2 build
	cmd := exec.CommandContext(ctx, a.tectonicBin, "-X", "build")
	cmd.Dir = wp.outDir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tectonic build failed: %v", err)
	}

	// 7. Read the generated PDF
	pdfPath := filepath.Join(wp.outDir, "build", "texput", "texput.pdf")
	return os.ReadFile(pdfPath)
}

func (a *LaTeXAssembler) extractOrGenerateBarcodes(payload []byte, dir string) error {
	parsed := gjson.ParseBytes(payload)
	var errResult error
	barcodesPresent := make(map[string]bool)
	parsed.ForEach(func(key, value gjson.Result) bool {
		k := key.String()
		if strings.HasPrefix(k, "barcode") {
			data, err := base64.StdEncoding.DecodeString(value.String())
			if err != nil {
				errResult = fmt.Errorf("invalid base64 for %s: %v", k, err)
				return false
			}
			path := filepath.Join(dir, k+".png")
			if err := os.WriteFile(path, data, 0644); err != nil {
				errResult = fmt.Errorf("failed to write %s: %v", path, err)
				return false
			}
			barcodesPresent[k] = true
		}
		return true
	})
	if errResult != nil {
		return errResult
	}

	if a.BarcodeRend != nil {
		if !barcodesPresent["barcodeZippeeawb"] && parsed.Get("zippeeAwb").Exists() {
			val := parsed.Get("zippeeAwb").String()
			data, err := barcode.RenderBarcodePNG(a.BarcodeRend, val)
			if err != nil {
				return fmt.Errorf("failed to generate barcodeZippeeawb: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "barcodeZippeeawb.png"), data, 0644); err != nil {
				return fmt.Errorf("failed to write generated barcodeZippeeawb: %v", err)
			}
		}
		if !barcodesPresent["barcodeRefcode"] && parsed.Get("referenceCode").Exists() {
			val := parsed.Get("referenceCode").String()
			data, err := barcode.RenderBarcodePNG(a.BarcodeRend, val)
			if err != nil {
				return fmt.Errorf("failed to generate barcodeRefcode: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "barcodeRefcode.png"), data, 0644); err != nil {
				return fmt.Errorf("failed to write generated barcodeRefcode: %v", err)
			}
		}
	}

	return nil
}
