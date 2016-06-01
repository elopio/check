package check

import (
	"fmt"
	"io"
	"sync"
)

type testReporter interface {
	Report(event, *C)
}

type event int

const (
	startTest event = iota
	failure
	panicked
	success
	expectedFailure
	skip
	missed
)

// -----------------------------------------------------------------------
// Output writer manages atomic output writing according to settings.

type outputWriter struct {
	m                    sync.Mutex
	writer               io.Writer
	wroteCallProblemLast bool
	verbosity            uint8
}

func newOutputWriter(writer io.Writer, verbosity uint8) *outputWriter {
	return &outputWriter{writer: writer, verbosity: verbosity}
}

func (ow *outputWriter) Report(e event, c *C) {
	switch e {
	case startTest:
		ow.writeStartTest(c)
	case failure:
		ow.writeProblem("FAIL", c)
	case panicked:
		ow.writeProblem("PANIC", c)
	case success:
		ow.writeSuccess("PASS", c)
	case expectedFailure:
		ow.writeSuccess("FAIL EXPECTED", c)
	case skip:
		ow.writeSuccess("SKIP", c)
	case missed:
		ow.writeSuccess("MISS", c)
	}
}

func (ow *outputWriter) writeStartTest(c *C) {
	if ow.verbosity > 1 {
		header := renderCallHeader("START", c, "", "\n")
		ow.m.Lock()
		ow.writer.Write([]byte(header))
		ow.m.Unlock()
	}
}

func (ow *outputWriter) writeProblem(label string, c *C) {
	var prefix string
	if ow.verbosity < 2 {
		prefix = "\n-----------------------------------" +
			"-----------------------------------\n"
	}
	header := renderCallHeader(label, c, prefix, "\n\n")
	ow.m.Lock()
	ow.wroteCallProblemLast = true
	ow.writer.Write([]byte(header))
	if ow.verbosity < 2 {
		c.logb.WriteTo(ow.writer)
	}
	ow.m.Unlock()
}

func (ow *outputWriter) writeSuccess(label string, c *C) {
	if ow.verbosity > 1 || (ow.verbosity == 1 && c.kind == testKd) {
		// TODO Use a buffer here.
		var suffix string
		if c.reason != "" {
			suffix = " (" + c.reason + ")"
		}
		if c.status() == succeededSt {
			suffix += "\t" + c.timerString()
		}
		suffix += "\n"
		if ow.verbosity > 1 {
			suffix += "\n"
		}
		header := renderCallHeader(label, c, "", suffix)
		ow.m.Lock()
		// Resist temptation of using line as prefix above due to race.
		if ow.verbosity < 2 && ow.wroteCallProblemLast {
			header = "\n-----------------------------------" +
				"-----------------------------------\n" +
				header
		}
		ow.wroteCallProblemLast = false
		ow.writer.Write([]byte(header))
		ow.m.Unlock()
	}
}

func renderCallHeader(label string, c *C, prefix, suffix string) string {
	pc := c.method.PC()
	return fmt.Sprintf("%s%s: %s: %s%s", prefix, label, niceFuncPath(pc),
		niceFuncName(pc), suffix)
}
