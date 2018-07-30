package bite

import (
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

type atomicSpinner struct {
	*spinner.Spinner
	state uint32 // 0 stopped or not started, 1 started, 3 disabled (stops if started and can't be started again)
}

func (s *atomicSpinner) Disable() {
	if s == nil {
		return
	}

	s.Stop()
	atomic.StoreUint32(&s.state, 3)
}

func (s *atomicSpinner) Start() {
	if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
		s.Spinner.Start()
	}
}

func (s *atomicSpinner) Stop() {
	if s == nil {
		return
	}

	if atomic.CompareAndSwapUint32(&s.state, 1, 0) {
		s.Spinner.Stop()
	}
}

func (s *atomicSpinner) Active() bool {
	return atomic.LoadUint32(&s.state) == 1
}

func (s *atomicSpinner) TryStart(timeToWaitBeforeSpin time.Duration) {
	go func() {
		timeToStartBePattient := time.Now().Add(timeToWaitBeforeSpin)
		for {
			// wait a bit bofore the checks.
			time.Sleep(100 * time.Millisecond)

			// if we didn't got any results after 'x' time, display the motion spinner.
			if time.Now().After(timeToStartBePattient) {
				s.Start()
				break
			}
		}
	}()
}

func newSpinner() *atomicSpinner {
	return &atomicSpinner{Spinner: spinner.New(spinner.CharSets[36], 150*time.Millisecond)}
}

type commandWriter struct {
	writer  io.Writer
	spinner *atomicSpinner
}

func (w *commandWriter) Write(b []byte) (int, error) {
	// Disable the spinner:
	// stop before the first output if started
	// and don't allow it to start even after the command finished, even if not started at all.
	//
	// Remember the spinner writes on `os.Stdout`
	// but commands through here.
	w.spinner.Disable()

	return w.writer.Write(b)
}

// ExecuteWithSpinner will make the spinner visible until first output from a command.
func ExecuteWithSpinner(cmd *cobra.Command) error {
	for _, a := range os.Args[1:] {
		// add the flag here so we avoid "unknown flag" errors, we don't actual need it now.
		// the "magic" with it is that this flag is not registered (on commands) or be visible (on help) until it's actually used.
		skipSpinner := a == "--no-spinner"
		switch a {
		case "help", "--help", "-h", "version", "--version":
			skipSpinner = true
		}

		if skipSpinner {
			_ = cmd.PersistentFlags().Bool("no-spinner", false, "disable the spinner")

			// if disabled, run the command's `Execute` as soon as possible.
			return cmd.Execute()
		}
	}

	spin := newSpinner()

	cmd.SetOutput(&commandWriter{
		writer:  cmd.OutOrStdout(),
		spinner: spin,
	})

	spin.TryStart(3 * time.Second)

	err := cmd.Execute()
	spin.Stop()
	return err
}
