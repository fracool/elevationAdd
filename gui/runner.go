package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
)

// globalPythonBin is set by checkPythonEnv and reused by Runner.
var globalPythonBin = "python3"

// extractedScriptPath holds the path to the embedded ele.py after first extraction.
var extractedScriptPath string

// RunArgs holds all parameters for a single processing run.
type RunArgs struct {
	GPXPath    string
	RasterDir  string
	OutputPath string
	Densify    bool
	MaxSpacing float64
}

// Runner executes ele.py as a subprocess and streams its output to the UI.
type Runner struct {
	state    *AppState
	cancelFn context.CancelFunc
	mu       sync.Mutex
}

// extractScript writes the embedded ele.py to a temp file (once) and returns its path.
// The file persists for the lifetime of the process.
func extractScript() (string, error) {
	if extractedScriptPath != "" {
		return extractedScriptPath, nil
	}
	f, err := os.CreateTemp("", "ele-*.py")
	if err != nil {
		return "", fmt.Errorf("failed to create temp script: %w", err)
	}
	if _, err := f.Write(embeddedScript); err != nil {
		f.Close()
		return "", fmt.Errorf("failed to write temp script: %w", err)
	}
	f.Close()
	extractedScriptPath = f.Name()
	return extractedScriptPath, nil
}

// checkPythonEnv verifies Python 3 and required packages are available.
// Sets globalPythonBin on success.
func checkPythonEnv() error {
	candidates := []string{"python3", "python"}
	found := ""
	for _, c := range candidates {
		path, err := exec.LookPath(c)
		if err != nil {
			continue
		}
		out, err := exec.Command(path, "--version").CombinedOutput()
		if err == nil && strings.HasPrefix(string(out), "Python 3") {
			found = path
			break
		}
	}
	if found == "" {
		return fmt.Errorf(
			"Python 3 not found in PATH.\n\n" +
				"macOS:   brew install python3\n" +
				"Windows: Download from python.org and check 'Add to PATH'")
	}
	globalPythonBin = found

	missing := detectMissingPackages()
	if len(missing) > 0 {
		return fmt.Errorf(
			"Missing Python packages: %s\n\nInstall with:\n  pip install %s",
			strings.Join(missing, ", "), strings.Join(missing, " "))
	}
	return nil
}

func detectMissingPackages() []string {
	pkgs := []string{"gpxpy", "rasterio", "pyproj", "geopy"}
	var missing []string
	for _, pkg := range pkgs {
		cmd := exec.Command(globalPythonBin, "-c", "import "+pkg)
		if err := cmd.Run(); err != nil {
			missing = append(missing, pkg)
		}
	}
	return missing
}

// Run executes ele.py with the given args. Must be called from a goroutine,
// not the UI thread. UI updates are dispatched via fyne.Do.
func (r *Runner) Run(args RunArgs) {
	scriptPath, err := extractScript()
	if err != nil {
		r.appendLog("[ERROR] " + err.Error())
		fyne.Do(func() { r.state.onRunComplete(false) })
		return
	}

	cmdArgs := []string{
		scriptPath,
		"--folder", args.RasterDir,
		"--gpx_file", args.GPXPath,
		"--output_file", args.OutputPath,
	}
	if args.Densify {
		cmdArgs = append(cmdArgs,
			"--densify",
			"--max_spacing", fmt.Sprintf("%.2f", args.MaxSpacing))
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancelFn = cancel
	r.mu.Unlock()

	cmd := exec.CommandContext(ctx, globalPythonBin, cmdArgs...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		r.appendLog("[ERROR] " + err.Error())
		fyne.Do(func() { r.state.onRunComplete(false) })
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		r.appendLog("[ERROR] " + err.Error())
		fyne.Do(func() { r.state.onRunComplete(false) })
		return
	}

	r.appendLog(fmt.Sprintf("$ %s %s\n", globalPythonBin, strings.Join(cmdArgs, " ")))

	if err := cmd.Start(); err != nil {
		r.appendLog("[ERROR] Failed to start: " + err.Error())
		fyne.Do(func() { r.state.onRunComplete(false) })
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	stream := func(pipe io.Reader, prefix string) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			r.appendLog(prefix + line)
		}
	}
	go stream(stdoutPipe, "")
	go stream(stderrPipe, "")

	go func() {
		defer cancel()
		wg.Wait()
		runErr := cmd.Wait()
		success := runErr == nil
		if !success && ctx.Err() == context.Canceled {
			r.appendLog("\n[CANCELLED] Process was cancelled.")
		} else if !success {
			r.appendLog("\n[ERROR] Process exited: " + runErr.Error())
		} else {
			r.appendLog("\n[DONE] Output written to: " + args.OutputPath)
		}
		fyne.Do(func() { r.state.onRunComplete(success) })
	}()
}

// Cancel terminates the running subprocess.
func (r *Runner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancelFn != nil {
		r.cancelFn()
	}
}

// appendLog appends a line to the UI log widget, thread-safely.
func (r *Runner) appendLog(text string) {
	fyne.Do(func() {
		cur := r.state.logWidget.Text
		if cur != "" {
			cur += "\n"
		}
		r.state.logWidget.SetText(cur + text)
		// Scroll to bottom by moving cursor to end
		lines := strings.Count(r.state.logWidget.Text, "\n")
		r.state.logWidget.CursorRow = lines
		r.state.logWidget.Refresh()
	})
}
