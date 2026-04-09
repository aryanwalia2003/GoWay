package assembler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"awb-gen/internal/barcode"

	"github.com/tidwall/gjson"
)

type warmProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	outDir string
}

type LaTeXAssembler struct {
	tectonicBin  string
	templateDir  string
	warmPool     chan *warmProcess
	HardCap      time.Duration
	BarcodeRend  barcode.Renderer
	barcodeCache *barcode.Cache
}

func NewLaTeXAssembler(bin, dir string, maxWorkers int) *LaTeXAssembler {
	absBin, _ := filepath.Abs(bin)
	absDir, _ := filepath.Abs(dir)
	a := &LaTeXAssembler{
		tectonicBin:  absBin,
		templateDir:  absDir,
		warmPool:     make(chan *warmProcess, maxWorkers),
		HardCap:      60 * time.Millisecond,
		BarcodeRend:  barcode.NewCode128Encoder(),
		barcodeCache: barcode.NewCache(),
	}

	for i := 0; i < maxWorkers; i++ {
		a.warmPool <- a.spawnWarmProcess()
	}

	return a
}

func (a *LaTeXAssembler) spawnWarmProcess() *warmProcess {
	outDir, err := os.MkdirTemp("", "tectonic_out")
	if err != nil {
		// On extreme failure, proceed with empty string, will fail loud later
		outDir = ""
	}

	cmd := exec.Command(a.tectonicBin, "--outdir", outDir, "-")
	cmd.Dir = outDir
	stdin, _ := cmd.StdinPipe()
	cmd.Start()

	return &warmProcess{
		cmd:    cmd,
		stdin:  stdin,
		outDir: outDir,
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
	src := filepath.Join(a.templateDir, templateID+".tex")
	content, err := os.ReadFile(src)
	if err != nil {
		wp.cmd.Process.Kill()
		wp.cmd.Wait()
		return nil, err
	}

	macros := ""
	if len(load) > 0 {
		macros, _ = MapToMacros(load)
		if err := a.extractOrGenerateBarcodes(load, wp.outDir); err != nil {
			wp.cmd.Process.Kill()
			wp.cmd.Wait()
			return nil, fmt.Errorf("failed to extract or generate barcodes: %v", err)
		}
	}
	combined := macros + "\n" + string(content)

	errCh := make(chan error, 1)
	go func() {
		io.WriteString(wp.stdin, combined)
		wp.stdin.Close()
		errCh <- wp.cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		wp.cmd.Process.Kill()
		<-errCh // wait for process to finish exiting
		return nil, ctx.Err()
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("tectonic failed: %v", err)
		}
	}

	pdfPath := filepath.Join(wp.outDir, "texput.pdf")
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
			var data []byte
			if cached, ok := a.barcodeCache.Get(val); ok {
				data = cached
			} else {
				var err error
				data, err = barcode.RenderBarcodePNG(a.BarcodeRend, val)
				if err != nil {
					return fmt.Errorf("failed to generate barcodeZippeeawb: %v", err)
				}
				a.barcodeCache.Set(val, data)
			}
			if err := os.WriteFile(filepath.Join(dir, "barcodeZippeeawb.png"), data, 0644); err != nil {
				return fmt.Errorf("failed to write generated barcodeZippeeawb: %v", err)
			}
		}
		if !barcodesPresent["barcodeRefcode"] && parsed.Get("referenceCode").Exists() {
			val := parsed.Get("referenceCode").String()
			var data []byte
			if cached, ok := a.barcodeCache.Get(val); ok {
				data = cached
			} else {
				var err error
				data, err = barcode.RenderBarcodePNG(a.BarcodeRend, val)
				if err != nil {
					return fmt.Errorf("failed to generate barcodeRefcode: %v", err)
				}
				a.barcodeCache.Set(val, data)
			}
			if err := os.WriteFile(filepath.Join(dir, "barcodeRefcode.png"), data, 0644); err != nil {
				return fmt.Errorf("failed to write generated barcodeRefcode: %v", err)
			}
		}
	}

	return nil
}
