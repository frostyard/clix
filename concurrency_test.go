package clix

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
)

// guardedWriter is a mutex-guarded io.Writer wrapping a bytes.Buffer. It is
// the shape clix documents for a consumer that calls the output paths from
// several goroutines: clix adds no synchronization of its own, so the
// consumer's writer must serialize its own writes.
//
// This type is deliberately the only synchronization in the test. Replacing
// it with a bare *bytes.Buffer makes `go test -race` report a data race,
// which is what proves the Race Detection CI job is no longer vacuous.
type guardedWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *guardedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

// snapshot returns a copy of everything written so far.
func (w *guardedWriter) snapshot() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return bytes.Clone(w.buf.Bytes())
}

// TestOutputPathsUnderConcurrentUse drives OutputJSON, OutputJSONError, and
// NewReporter from several goroutines at once against a serialized writer.
// It pins the property that clix's output paths hold no shared mutable state
// of their own beyond the documented Stdout / Stderr seams and flag globals:
// every record must arrive whole, and the race detector must stay quiet.
//
// It deliberately does not call t.Parallel() and does not swap os.Stdout:
// both Stdout / Stderr and the flag globals are package-level state, and
// os.Stdout is process-wide (see docs/design/overview.md, "Test isolation").
func TestOutputPathsUnderConcurrentUse(t *testing.T) {
	const (
		goroutines = 8
		iterations = 4
	)

	w := &guardedWriter{}

	prevJSON, prevSilent := JSONOutput, Silent
	Stdout, Stderr = w, w
	JSONOutput, Silent = true, false
	t.Cleanup(func() {
		Stdout, Stderr = nil, nil
		JSONOutput, Silent = prevJSON, prevSilent
	})

	// Each goroutine records its own findings in its own slot, so the test's
	// bookkeeping introduces no sharing of its own.
	type outcome struct {
		notWritten int // OutputJSON returned false
		encodeErr  int // OutputJSON returned a non-nil error
		nilError   int // OutputJSONError returned nil
		nilReport  int // NewReporter returned nil
	}
	outcomes := make([]outcome, goroutines)

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := &outcomes[g]
			for i := range iterations {
				written, err := OutputJSON(map[string]any{
					"goroutine": g,
					"iteration": i,
					"kind":      "data",
				})
				if !written {
					got.notWritten++
				}
				if err != nil {
					got.encodeErr++
				}

				if e := OutputJSONError("concurrent failure", errors.New("boom")); e == nil {
					got.nilError++
				}

				if r := NewReporter(); r == nil {
					got.nilReport++
				}
			}
		}()
	}
	wg.Wait()

	for g, got := range outcomes {
		if got.notWritten != 0 {
			t.Errorf("goroutine %d: OutputJSON returned written=false %d time(s), want 0", g, got.notWritten)
		}
		if got.encodeErr != 0 {
			t.Errorf("goroutine %d: OutputJSON returned a non-nil error %d time(s), want 0", g, got.encodeErr)
		}
		if got.nilError != 0 {
			t.Errorf("goroutine %d: OutputJSONError returned nil %d time(s), want 0", g, got.nilError)
		}
		if got.nilReport != 0 {
			t.Errorf("goroutine %d: NewReporter returned nil %d time(s), want 0", g, got.nilReport)
		}
	}

	// Every OutputJSON call and every OutputJSONError envelope is one
	// top-level JSON value; none may be interleaved or truncated.
	wantValues := goroutines * iterations * 2
	dec := json.NewDecoder(bytes.NewReader(w.snapshot()))
	values := 0
	for {
		var v map[string]any
		err := dec.Decode(&v)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decoding value %d: %v", values+1, err)
		}
		values++
	}
	if values != wantValues {
		t.Errorf("decoded %d complete top-level JSON values, want %d", values, wantValues)
	}
}
