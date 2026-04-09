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

type LaTeXAssembler struct {
	tectonicBin string
	templateDir string
	sem         chan struct{}
	HardCap     time.Duration
	BarcodeRend barcode.Renderer
}

func NewLaTeXAssembler(bin, dir string, maxWorkers int) *LaTeXAssembler {
	absBin, _ := filepath.Abs(bin)
	absDir, _ := filepath.Abs(dir)
	return &LaTeXAssembler{
		tectonicBin: absBin,
		templateDir: absDir,
		sem:         make(chan struct{}, maxWorkers),
		HardCap:     60 * time.Millisecond,
		BarcodeRend: barcode.NewCode128Encoder(),
	}
}

func (a *LaTeXAssembler) Assemble(ctx context.Context, templateID string, payload []byte) ([]byte, error) {
	a.sem <- struct{}{}
	defer func() { <-a.sem }()

	ctx, cancel := context.WithTimeout(ctx, a.HardCap)
	defer cancel()

	outDir, err := os.MkdirTemp("", "tectonic_out")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	return a.compileTemplate(ctx, templateID, payload, outDir)
}

func (a *LaTeXAssembler) compileTemplate(ctx context.Context, templateID string, load []byte, out string) ([]byte, error) {
	src := filepath.Join(a.templateDir, templateID+".tex")
	content, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}

	macros := ""
	if len(load) > 0 {
		macros, _ = MapToMacros(load)
		if err := a.extractOrGenerateBarcodes(load, out); err != nil {
			return nil, fmt.Errorf("failed to extract or generate barcodes: %v", err)
		}
	}
	combined := macros + "\n" + string(content)

	cmd := exec.CommandContext(ctx, a.tectonicBin, "--outdir", out, "-")
	cmd.Dir = out // Ensure LaTeX can find relative image paths
	cmd.Stdin = strings.NewReader(combined)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("tectonic failed: %v\nOut: %s", err, string(outBytes))
	}

	pdfPath := filepath.Join(out, "texput.pdf")
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
