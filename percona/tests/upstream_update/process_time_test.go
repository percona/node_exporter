package upstream_update

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestCpuTime(t *testing.T) {
	doTestStats(t, 10, 25)
}

func doTestStats(t *testing.T, cnt int, size int) {
	var a []int64

	for i := 0; i < cnt; i++ {
		d, _ := doTest(t, size)
		a = append(a, d)
	}

	var ba []float64

	for _, f := range a {
		ba = append(ba, float64(f))
	}
	data := ba
	med, _ := stats.Mean(data)
	dev, _ := stats.StandardDeviation(data)

	fmt.Printf("loop %d: sample time: %.2f [mean: %.2f; dev: %.2f]\n", size, med/float64(size), med, dev)
}

func doTest(t *testing.T, iterations int) (int64, error) {
	lines, err := os.ReadFile("test.exporter-flags.txt")
	if !assert.NoError(t, err, "unable to read exporter args file") {
		return 0, err
	}

	linesStr := string(lines)
	linesStr += "--web.listen-address=127.0.0.1:20001"
	linesArr := strings.Split(linesStr, "\n")

	fileName := "../../../node_exporter"
	cmd := exec.Command(fileName, linesArr...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Start()

	if !assert.NoError(t, err, "Failed to start exporter. Process output:\n%q", out.String()) {
		return 0, err
	}

	waitForExporter()

	total1 := getCPUSample(cmd.Process.Pid)

	for i := 0; i < iterations; i++ {
		getMetrics(t)
		time.Sleep(1 * time.Millisecond)
	}

	total2 := getCPUSample(cmd.Process.Pid)

	err = cmd.Process.Signal(unix.SIGTERM)
	assert.NoError(t, err, "Failed to send SIGTERM to exporter process")

	err = cmd.Wait()
	assert.NoError(t, err, "Failed to wait for exporter process termination")

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
			//fmt.Println(line)
			//fmt.Printf("%s; %s; %s; %s\n", fields[13], fields[14], fields[15], fields[16])

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

			//fmt.Printf("cpu ticks: %d; cpu time: %.2f\n", totalTime, float64(totalTime)/float64(hertz))

			total = totalTime

			return
		}
	}
	return
}

func getMetrics(t *testing.T) {
	resp, err := http.Get("http://127.0.0.1:20001/metrics")
	if !assert.NoError(t, err, "Failed to get response from exporters web interface") {
		return
	}

	assert.Equal(t, resp.StatusCode, 200, "Response fail")
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err, "Failed to get response from exporters web interface")
		bodyString := string(bodyBytes)
		assert.NotEmpty(t, bodyString)
	}
}

func waitForExporter() {
	for !doGet() {
		time.Sleep(1 * time.Millisecond)
	}
}

func doGet() bool {
	resp, err := http.Get("http://127.0.0.1:20001/metrics")
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
