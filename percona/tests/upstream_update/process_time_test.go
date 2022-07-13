package upstream_update

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/stretchr/testify/assert"
	"github.com/tklauser/go-sysconf"
	"golang.org/x/sys/unix"
)

const (
	exporterWaitTimeoutMs = 1000 // time to wait for exporter process start

	portRangeStart = 20000 // exporter web interface listening port
	portRangeEnd   = 20100 // exporter web interface listening port
)

func TestCpuTime(t *testing.T) {
	doTestStats(t, 5, 25)
}

func doTestStats(t *testing.T, cnt int, size int) {
	var durations []float64

	for i := 0; i < cnt; i++ {
		d, _ := doTest(t, size)
		durations = append(durations, float64(d))
	}

	mean, _ := stats.Mean(durations)
	stdDev, _ := stats.StandardDeviation(durations)
	stdDev = float64(100) / mean * stdDev

	clockTicks, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	if err != nil {
		panic(err)
	}

	mean = mean / float64(clockTicks) * float64(1000)

	fmt.Printf("loop %dx%d: sample time: %.2f ms [std dev: %.1f %%]\n", cnt, size, mean/float64(size), stdDev)
}

func checkPort(port int) bool {
	ln, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		return false
	}

	_ = ln.Close()
	return true
}

func doTest(t *testing.T, iterations int) (int64, error) {
	lines, err := os.ReadFile("test.exporter-flags.txt")
	if !assert.NoError(t, err, "unable to read exporter args file") {
		return 0, err
	}

	var port = -1
	for i := portRangeStart; i < portRangeEnd; i++ {
		if checkPort(i) {
			port = i
			break
		}
	}

	if port == -1 {
		panic(fmt.Sprintf("Failed to find free port in range [%d..%d]", portRangeStart, portRangeEnd))
	}

	linesStr := string(lines)
	linesStr += fmt.Sprintf("--web.listen-address=127.0.0.1:%d", port)
	linesArr := strings.Split(linesStr, "\n")

	fileName := "../../../node_exporter"
	cmd := exec.Command(fileName, linesArr...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Start()
	if !assert.NoError(t, err, "Failed to start exporter. Process output:\n%q", out.String()) {
		return 0, err
	}

	waitForExporter(port)

	total1 := getCPUTime(cmd.Process.Pid)

	for i := 0; i < iterations; i++ {
		err = tryGetMetrics(port)
		if !assert.NoError(t, err) {
			return 0, err
		}

		time.Sleep(1 * time.Millisecond)
	}

	total2 := getCPUTime(cmd.Process.Pid)

	err = cmd.Process.Signal(unix.SIGINT)
	assert.NoError(t, err, "Failed to send SIGINT to exporter process")

	err = cmd.Wait()
	if err != nil && err.Error() != "signal: interrupt" {
		assert.NoError(t, err, "Failed to wait for exporter process termination. Process output:\n%q", out.String())
		panic(err)
	}

	return total2 - total1, nil
}

func getCPUTime(pid int) (total int64) {
	contents, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		numFields := len(fields)
		if numFields > 3 {
			i, err := strconv.ParseInt(fields[13], 10, 64)
			if err != nil {
				panic(err)
			}

			totalTime := i

			i, err = strconv.ParseInt(fields[14], 10, 64)
			if err != nil {
				panic(err)
			}

			totalTime += i

			total = totalTime

			return
		}
	}
	return
}

func tryGetMetrics(port int) error {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if err != nil {
		return fmt.Errorf("failed to get response from exporters web interface: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get response from exporters web interface: %w", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response from exporters web interface: %w", err)
	}

	bodyString := string(bodyBytes)
	if bodyString == "" {
		return fmt.Errorf("got empty response from exporters web interface: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to close response body: %w", err)
	}

	return nil
}

func waitForExporter(port int) {
	watchdog := exporterWaitTimeoutMs

	for ; tryGetMetrics(port) != nil && watchdog > 0; watchdog-- {
		time.Sleep(1 * time.Millisecond)
	}

	if watchdog == 0 {
		panic(fmt.Sprintf("Failed to wait for exporter (on port %d)", port))
	}
}
