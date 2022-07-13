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
	for i := 20000; i < 20100; i++ {
		if checkPort(i) {
			port = i
			break
		}
	}

	if port == -1 {
		panic("Failed to find free port in range [20000..20100]")
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

	total1 := getCPUSample(cmd.Process.Pid)

	for i := 0; i < iterations; i++ {
		getMetrics(t, port)
		time.Sleep(1 * time.Millisecond)
	}

	total2 := getCPUSample(cmd.Process.Pid)

	err = cmd.Process.Signal(unix.SIGINT)
	assert.NoError(t, err, "Failed to send SIGINT to exporter process")

	err = cmd.Wait()
	if err != nil && err.Error() != "signal: interrupt" {
		assert.NoError(t, err, "Failed to wait for exporter process termination. Process output:\n%q", out.String())
		panic(err)
	}

	return total2 - total1, nil
}

func getCPUSample(pid int) (total int64) {
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

func getMetrics(t *testing.T, port int) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if !assert.NoError(t, err, "Failed to get response from exporters web interface") {
		return
	}

	assert.Equal(t, resp.StatusCode, 200, "Response fail")
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		assert.NoError(t, err, "Failed to close response from exporters web interface")
	}(resp.Body)

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err, "Failed to get response from exporters web interface")
		bodyString := string(bodyBytes)
		assert.NotEmpty(t, bodyString)
	}
}

func waitForExporter(port int) {
	watchdog := 1000
	for ; !doGet(port) && watchdog > 0; watchdog-- {
		time.Sleep(1 * time.Millisecond)
	}

	if watchdog == 0 {
		panic(fmt.Sprintf("Failed to wait for exporter (on port %d)", port))
	}
}

func doGet(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}

		bodyString := string(bodyBytes)
		if bodyString == "" {
			return false
		}
	}

	return true
}
