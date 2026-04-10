package assembler

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"awb-gen/internal/barcode"

	"github.com/tidwall/gjson"
)

type warmProcess struct {
	outDir string
	srcDir string
}

type LaTeXAssembler struct {
	tectonicBin   string
	templateDir   string
	warmPool      chan *warmProcess
	HardCap       time.Duration
	BarcodeRend   barcode.Renderer
	templateMu    sync.RWMutex
	templateCache map[string][2][]byte
}

func NewLaTeXAssembler(bin, dir string, maxWorkers int) *LaTeXAssembler {
	absBin, _ := filepath.Abs(bin)
	absDir, _ := filepath.Abs(dir)
	a := &LaTeXAssembler{
		tectonicBin:   absBin,
		templateDir:   absDir,
		warmPool:      make(chan *warmProcess, maxWorkers),
		HardCap:       5 * time.Second,
		BarcodeRend:   barcode.NewCode128Encoder(),
		templateCache: make(map[string][2][]byte),
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

func (a *LaTeXAssembler) loadTemplateBytes(templateID string) (preamble, body []byte, err error) {
	a.templateMu.RLock()
	if v, ok := a.templateCache[templateID]; ok {
		a.templateMu.RUnlock()
		return v[0], v[1], nil
	}
	a.templateMu.RUnlock()

	pb, err := os.ReadFile(filepath.Join(a.templateDir, templateID+"_preamble.tex"))
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	if len(pb) == 0 {
		pb = []byte("")
	}
	bd, err := os.ReadFile(filepath.Join(a.templateDir, templateID+".tex"))
	if err != nil {
		return nil, nil, err
	}

	a.templateMu.Lock()
	a.templateCache[templateID] = [2][]byte{pb, bd}
	a.templateMu.Unlock()
	return pb, bd, nil
}

func resetWorkDir(wp *warmProcess) error {
	toRemove := []string{
		filepath.Join(wp.srcDir, "index.tex"),
		filepath.Join(wp.srcDir, "_preamble.tex"),
		filepath.Join(wp.srcDir, "barcodeZippeeawb.png"),
		filepath.Join(wp.srcDir, "barcodeRefcode.png"),
	}
	for _, p := range toRemove {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (a *LaTeXAssembler) Assemble(ctx context.Context, templateID string, payload []byte) ([]byte, error) {
	wp := <-a.warmPool
	defer func() {
		if err := resetWorkDir(wp); err != nil {
			os.RemoveAll(wp.outDir)
			wp = a.spawnWarmProcess()
		}
		a.warmPool <- wp
	}()

	ctx, cancel := context.WithTimeout(ctx, a.HardCap)
	defer cancel()

	return a.executeWarmProcess(ctx, wp, templateID, payload)
}

func (a *LaTeXAssembler) executeWarmProcess(ctx context.Context, wp *warmProcess, templateID string, load []byte) ([]byte, error) {
	preambleContent, bodyContent, err := a.loadTemplateBytes(templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %v", err)
	}

	// 4. Generate macros and barcodes
	macros := ""
	if len(load) > 0 {
		parsed := gjson.ParseBytes(load)
		macros, _ = MapParsedToMacros(parsed)
		if err := a.extractOrGenerateBarcodes(parsed, wp.srcDir); err != nil {
			return nil, fmt.Errorf("failed to extract or generate barcodes: %v", err)
		}
	}

	// In this "hybrid" mode, we combine everything into index.tex and run
	// tectonic directly on it, bypassing the Tectonic V2 "Project Mode" (-X build)
	// which had high overhead (~1.2s of project/bundle management).
	bufLength := len(preambleContent) + 1 + len(macros) + 1 + len(bodyContent)
	buf := make([]byte, 0, bufLength)
	buf = append(buf, preambleContent...)
	buf = append(buf, '\n')
	buf = append(buf, macros...)
	buf = append(buf, '\n')
	buf = append(buf, bodyContent...)

	indexFile := filepath.Join(wp.srcDir, "index.tex")
	if err := os.WriteFile(indexFile, buf, 0644); err != nil {
		return nil, fmt.Errorf("failed to write index.tex: %v", err)
	}

	// 6. Execute Tectonic V1-style (direct file compilation)
	cmd := exec.CommandContext(ctx, a.tectonicBin, "-C", indexFile, "--outdir", wp.outDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("tectonic build failed: %v, output: %s", err, string(out))
	}

	// 7. Read the generated PDF
	pdfPath := filepath.Join(wp.outDir, "index.pdf")
	return os.ReadFile(pdfPath)
}

func (a *LaTeXAssembler) extractOrGenerateBarcodes(parsed gjson.Result, dir string) error {
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
