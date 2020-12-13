package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)
var start = time.Now()
var format  = flag.String("format", "default", "timestamp format")
var verbose = flag.Bool("verbose", false, "verbose output")
var tabs = flag.Bool("tabs", false, "use tabs rather than spaces after the timestamp")
var utc = flag.Bool("utc", false, "use utc timestamps instead of localtime ones.")
var millis = flag.Bool("millis", false, "calculate timestamps in milliseconds since program start.")

type TimeFormat int
const (
	DEFAULT TimeFormat = iota
	ANSI
	RFC3339
	RFC3339Nano
)

func (tf *TimeFormat) String() string {
	var res string

	switch *tf {
	case DEFAULT: res = "2006/01/02 03:04:05"
	case ANSI: res = time.ANSIC
	case RFC3339: res = time.RFC3339
	case RFC3339Nano: res = time.RFC3339Nano
	default: log.Panicf("Unexpected")
	}

	return res
}

func (tf *TimeFormat) fromString(s *string) bool {
	res := true

	switch *s {
	case "default": *tf = DEFAULT
	case "ansi": *tf = ANSI
	case "rfc3339": *tf = RFC3339
	case "rfc3339nano": *tf = RFC3339Nano
	default: res = false
	}

	return res
}

// TimestampedWriter is a writer that splits text on newlines and outputs lines one at the time, prepending each
// with a timestamp.
type TimestampedWriter struct {
	writer     io.Writer
	format     string
	utc        bool
	millis     bool
	tabs       bool
	incomplete []byte
}

// NewTimestampedWriter creates a new TimestampedWriter
func NewTimestampedWriter(w io.Writer, timeFormat TimeFormat, utc *bool, millis *bool, tabs *bool) *TimestampedWriter {
	return &TimestampedWriter{
		writer:     w,
		format:     timeFormat.String(),
		utc:        *utc,
		millis:     *millis,
		tabs:       *tabs,
		incomplete: make([]byte, 0),
	}
}

func (tsw *TimestampedWriter) Write(p []byte)(int, error) {
	lines := bytes.Split(p, []byte("\n"))
	last := lines[len(lines) -1]

	for _, line := range lines[:len(lines) -1] {
		var (
			timestamp string
			err error
		)

		now := time.Now()
		if *millis {
			timestamp = fmt.Sprintf("%12.3fms", float64(now.Sub(start).Microseconds()) / 1000)
		} else {
			if *utc {
				now = now.UTC()
			}
			timestamp = now.Format(tsw.format)
		}
		_, err = tsw.writer.Write([]byte(timestamp)); if err != nil {
			return 0, err
		}

		var sep = "| "
		if tsw.tabs {
			sep = "|\t"
		}
		_, err = tsw.writer.Write([]byte(sep)); if err != nil {
			return 0, err
		}

		if 0 < len(tsw.incomplete) {
			_, err = tsw.writer.Write(tsw.incomplete); if err != nil {
				return 0, err
			}
		}
		tsw.incomplete = last

		_, err = tsw.writer.Write(line); if err != nil {
			return 0, err
		}

		_, err = tsw.writer.Write([]byte("\n")); if err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

func execute(name string, args []string, tf TimeFormat) {
	var err error

	if *verbose {
		log.Printf("invoking command: %v, args: %v", name, args)
	}
	cmd := exec.Command(name, args...)

	stdoutIn, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("ERROR: could not connect to stdout pipe: %s", err)
	}

	stderrIn, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("ERROR: could not connect to stderr pipe: %s", err)
	}

	stdout := NewTimestampedWriter(os.Stdout, tf, utc, millis, tabs)
	stderr := NewTimestampedWriter(os.Stderr, tf, utc, millis, tabs)

	err = cmd.Start()
	if err != nil {
		log.Fatalf("ERROR: could not start: '%s'\n", err)
	}

	processStreams(stdout, stdoutIn, stderr, stderrIn)

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("ERROR: command failed: %s", err)
	}
}

func processStreams(stdout *TimestampedWriter, stdoutIn io.ReadCloser, stderr *TimestampedWriter, stderrIn io.ReadCloser) {
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		_, err := io.Copy(stdout, stdoutIn)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()

	go func() {
		_, err := io.Copy(stderr, stderrIn)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()

	wg.Wait()
}

func init() {
	/* timestamps in logging can easily get confused with output */
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	flag.CommandLine.Usage = func() {
		output := flag.CommandLine.Output()
		fmt.Fprintf(output,"ts - run a command with timestamped output\n")
		fmt.Fprintf(output,"usage:\n  ts [ options ] cmd args...\n")
		fmt.Fprintf(output,"options:\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if *millis && *utc {
		log.Printf("WARNING: -utc will be ignored when -millis is specified.")
	}
	var tf TimeFormat
	ok := tf.fromString(format); if ! ok {
		log.Fatal(fmt.Sprintf("illegal time format identifier: %v", *format))
	}

	cliArgs := flag.Args()
	if len(cliArgs) < 1 {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	name := cliArgs[0]
	args := cliArgs[1:]

	execute(name, args, tf)
}