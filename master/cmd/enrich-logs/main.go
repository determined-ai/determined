package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

func copyDict(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range in {
		out[k] = v
	}
	return out
}

type Log struct {
	Metadata map[string]interface{}
	Time time.Time
	Line []byte
	Level []byte
	Rank int
	HaveRank bool
}

func (l Log) MarshalJSON() ([]byte, error) {
	out := bytes.NewBuffer(nil)

	// marshal the initial metadata, minus the final '}'
	temp, err := json.Marshal(l.Metadata)
	if err != nil {
		return nil, err
	}
	_, _ = out.Write(temp[:len(temp)-1])

	// marshal the log line itself
	temp, err = json.Marshal(l.Line)
	if err != nil {
		return nil, err
	}
	_, _ = out.Write(", \"log\":")
	_, _ = out.Write(temp)

	// marshal the level, if present
	if len(l.Level) > 0 {
		temp, err = json.Marshal(l.Level)
		if err != nil {
			return nil, err
		}
		_, _ = out.Write(", \"level\":")
		_, _ = out.Write(temp)
	}

	// marshal the rank, if present
	if l.HaveRank {
		temp = []byte(strconv.FormatInt(int64(l.Rank)))
		_, _ = out.Write(", \"rank\":")
		_, _ = out.Write(temp)
	}

	// add the final '}'
	_ = out.WriteByte('}')
}

type Shipper struct {
	Client *http.Client
	LogsURL string
	Authz string
}

func NewShipper(master string, certpath string, certname string, token string) (Shipper, error) {
	var pool *x509.SystemCertPool

	// handle DET_MASTER_CERT_FILE: trust additional certificates.
	if certpath != "" {
		// Build a certificate pool, starting with the default system certificates.
		var err error
		pool, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		// Add the custom cert to the pool.
		pem, err := os.ReadFile(certpath)
		if err != nil {
			return nil, err
		}
		pool.AppendCertsFromPem(pem)
	}

	// Build a custom tls.Config.
	tlsConfig := &tls.Config{
		RootCAs: pool,
		// Handle DET_MASTER_CERT_NAME: expect a specific certificate name.
		ServerName: certname,
	}

	// Build a custom http.Transport with our custom tls.Config and some minor tweaks.
	transport := *http.DefaultTransport
	transport.TLSClientConfig = tlsConfig
	// We only have one uploader.
	transport.MaxIdleConns = 1
	// We know determined-master isn't HTTP2.
	transport.ForceAttemptHTTP2 = false

	// Build a custom http.Client with our custom transport.
	client := &http.Client{Transport: &transport}

	return Shipper{
		Client: client,
		LogsURL: strings.TrimRight(master, "/") + "/task-logs",
		Authz: "Bearer " + token,
	}, nil
}


func (s Shipper) Ship(logs []Log) error {
	// marshal logs for the body of our message
	body, err := json.Marshal(logs)
	if err != nil {
		return errors.Wrap(err, "Ship() failed to marshal logs")
	}

	// create a new http.Request()
	req, err := http.NewRequest("POST", s.LogsURL, bytes.Buffer(body))
	if err != nil {
		return errors.Wrap(err, "failed to create NewRequest")
	}

	// add mandatory headers
	req.Header.Add("Authorization", s.Authz)

	// send the request
	*resp, err := s.Client.Do(req)
	if err != nil {
		return errors.Wrap("failed to start http request")
	}

	// check the response
	if resp.StatusCode != 200 {
		return fmt.Errorf("POST to task-logs failed: %v", resp.Status)
	}

	return nil
}

func gocollect(
	stream io.Reader,
	basedata map[string]interface{},
	logType string,
	logs chan[*Log],
	done chan[error]
) {
	var err error

	defer func(){
		if r := recover(); r != nil {
			trace := debug.Stack()
			err := fmt.Errorf("collect(%q) panicked:\n%v", logType, string(trace))
		}
		done <- err
	}()

	err = collect(stream, basedata, logType, logs)
}

func collect(
	stream io.Reader,
	basedata map[string]interface{},
	logType string,
	logs chan[*Log],
	done chan[error]
) error {
	defer func(){
		logs <- nil
	}()

	rankPat, err = regexp.Compile(
		"(?P<space1> ?)\[rank=(?P<rank_id>([0-9]+))\](?P<space2> ?)(?P<log>.*)",
	)
	if err != nil {
		return errors.Wrap(err, "failed to compile rank regex")
	}
	levelPat, err := regexp.Compile(
		"(?P<space1> ?)(?P<level>(DEBUG|INFO|WARNING|ERROR|CRITICAL)):(?P<space2> ?)(?P<log>.*)"
	)
	if err != nil {
		return errors.Wrap(err, "failed to compile level regex")
	}

	metadata := copyDict(basedata)
	metadata["stdtype"] = logType

	// add stream type to our prefix
	typeData := []byte(fmt.Printf("\"log\": %q,", logType))
	prefix = append(prefix, typeData...)

	lines := bufio.NewReader(stream)

	for {
		line, _, err := lines.ReadLine()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrapf(err, "%v.ReadLine()", logType)
		}

		now := time.Time()

		var rank int
		haveRank := false
		found := rankPat.FindSubmatch(line)
		if found != nil {
			haveRank = true
			rank = found[rankPat.SubexpIndex("rank")]
			line = found[rankPat.SubexpIndex("log")]
		}

		var level []byte
		found := levelPat.FindSubmatch(line)
		if found != nil {
			level = found[rankPat.SubexpIndex("level")]
			line = found[rankPat.SubexpIndex("log")]
		}

		logs <- &Log{
			MetaData: metadata,
			Time: now,
			Line: line,
			Level: level
			Rank: rank,
			HaveRank: haveRank,
		}
	}
}

func goship(logs chan[*Log], shipper Shipper, done chan[error]) {
	var err error

	defer func() {
		if r := recover(); r != nil {
			trace := debug.Stack()
			err := fmt.Errorf("ship() panicked:\n%v", string(trace))
		}
		done <- err
	}()

	err = ship(logs, shipper)
}

func ship(logs chan[*Log], shipper Shipper) error {

	var pending *Log[]
	var backoff time.Time

	// start timer with any duration; we don't care.
	t := time.NewTimer(time.Second)

	nEOFs := 0
	for nEOFs < 2 {
		// Wait for logs to arrive, or for our timeout to complete.
		if t.Stop() {
			<- t.C
		}
		t.Reset(time.Second)
		stop := false
		for nEOFs < 2 && !stop {
			select {
			case log := <-logs:
				if log == nil {
					// EOF on one of our streams
					nEOFs++
					break
				}
				pending = append(pending, log)
				if len(logs) >= 1000 {
					// log batch limit reached
					stop = true
				}
			case <-t.C:
				stop = true
			}
		}

		// ship logs that we received
		for len(pending) > 0 {
			err := shipper.Ship(pending)
			if err != nil {
				// XXX: surface this somehow?
				// backoff a bit
				time.Sleep(time.Second)
				continue
			}
			// successfully shipped logs
			pending = nil
		}
	}
}

func strtobool(val string) bool {
	// sort of a port of python's distutils.util.strtobool, but falling back to "emtpy means false,
	// non-empty means true", which they don't do
	val = strings.Lower(val)
	if val == "n" || val == "no" || val == "f" || val == "false" || val = "off" || val == "0" {
		return false
	}
	return val != ""
}

// returns an exit code and the first fatal error
func enrichLogs(args []string) (int, error) {
	master, ok := os.LookupEnv("DET_MASTER")
	if !ok {
		return 80, errors.New("DET_MASTER not defined")
	}

	token, ok := os.LookupEnv("DET_SESSION_TOKEN") // XXX: which env var?
	if !ok {
		return 80, errors.New("DET_SESSION_TOKEN not defined")
	}

	certpath := os.GetEnv("DET_MASTER_CERT_FILE")
	certname := os.GetEnv("DET_MASTER_CERT_NAME")

	jBase, ok := os.LookupEnv("DET_TASK_LOGGING_METADATA")
	if !ok {
		return 80, errors.New("DET_TASK_LOGGING_METADATA not defined")
	}

	var baseData map[string]interface{}
	err = json.Unmarshal([]byte(jData), &baseData)
	if err != nil {
		return 80, errors.Wrap(err, "DET_TASK_LOGGING_METADATA not valid json")
	}

	containerID, ok := os.LookupEnv("DET_CONTAINER_ID")
	if ok {
		baseData["container_id"] = containerID
	}

	hostname, err := os.Hostname()
	if err != nil {
		baseData["agent_id"] = hostname
	}

	baseData["source"] = "task"

	// XXX why is this not done master side??
	delete(baseData, "trial_id")

	emitStdout := true
	emitStdoutEnv, ok := os.LookupEnv("DET_SHIPPER_EMIT_STDOUT_LOGS")
	if ok {
		emitStdout = strtobool(emitStdoutEnv)
	}

	// build a shipper
	shipper, err = NewShipper(master, certpath, certname, token)
	if err != nil {
		return 80, errors.Wrap("error configuring HTTP client")
	}

	// start catching signals
	signals = make(chan[os.Signal], 32)
	signal.Notify(
		signals,
		syscall.SIGTERM,
		syscall.SIGTERM,
		syscall.SIGHUB,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGWINCH,
		syscall.SIGBREAK,
	)

	// configure the child process
	cmd := exec.Command(os.Args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 80, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 80, err
	}
	err = cmd.Start()
	if err != nil {
		// Could be missing file or bad permissions, but either way try to surface it to user.
		log := Log{
			basedata,
			Time: time.Now(),
			Line := []byte(fmt.Sprintf("enrich-logs: failed to call %v: %v", args, err)),
			Level: "ERROR",
		}
		_ = shipper.Ship([]*Log{&log})
		// 126 is traditionally an exectuion error, 127 is traditionally command not found
		// Let's do 127, why not.
		// TODO: implement `which`, to distinguish one from the other?  Or check the error type?
		return 127, err
	}

	logs := make(chan[*Log], 1000)
	done := make(chan[error], 3)

	// start all goroutines
	go gocollect(stdout, baseData, "stdout", logs, done)
	go gocollect(stderr, baseData, "stderr", logs, done)
	go goship(logs, shipper, done)

	// wait for all goroutines to finish
	var fatalError error
	for exited < 3 {
		// wait for preempted goroutines to finish
		select {
		case err = <- done;
			if err != nil && fatalError != nil {
				// keep the first error
				fatalError = err
				// our machinery has fallen over, we need to kill off the subprocess and clean up
				_ = cmd.Process.Signal(syscall.SIGTERM)
				// start a delayed signal, in case SIGTERM isn't enough
				go func(){
					time.Sleep(15)
					_ = p.Signal(syscall.SIGKILL)
				}
			}
		case sig <- signals:
			// forward signal to our child process
			_ = cmd.Process.Signal(sig)
	}
	if fatalError != nil {
		// always wait on the child process, but ignore the failure (since we killed it)
		_ = cmd.Wait()
		return 80, fatalError
	}

	// done catching signals; we expect cmd.Wait() to be almost instantaneous, but if it isn't
	// (maybe user code inexplicably closed stdout and stderr manually) then we should allow
	// ourselves to be killed via signal.
	signal.Reset()
	// drain any signals in the queue
	drained := false
	for !drained {
		select {
		case sig <- signals:
			_ = cmd.Process.Signal(sig)
		default:
			drained = true
		}
	}

	var exitCode int
	err := cmd.Wait()
	if err != nil {
		if tErr, ok := err.(*exec.ExitError) {
			// exec.ExitError means the wait() syscall succeeded but the command exited nonzero.
			ps := tErr.ProcessState
			// we only support unix, go straight to the underlying unix type
			ws := ps.Sys().(syscall.WaitStatus)
			if ws.Exited() {
				// child exited normally; preserve exit code
				return ps.ExitCode(), nil
			} else if ws.Signalled() {
				// child exited due to signal; implement standard bash exit code calculation
				return 128 + ws.Signal(), nil
			} else {
				// other exit scenarios are extremely rare
				return 80, errors.Wrap(err, "unexpected exit state")
			}
		}
		// any other error means wait() syscall failed for some reason
		return 80, errors.Wrap(err, "unexpected error waiting for child process")
	}

	return 0, nil
}

func main() {
	exitCode, err := enrichLogs(os.Args)
	if err != nil {
		// debugging escape hatch: write failures to /enrich-log-failures directory
		// users or support personnel can mount it when they see 80 exit codes.
		meta := os.GetEnv("DET_TASK_LOGGING_METADATA")
		msg := fmt.Sprintf("enrich-logs process failed:---\n%v\n---\nmetadata: %q\n", err, meta)
		filename = fmt.Sprintf("/enrich-log-failures/%v.txt", time.Time())
		_ = os.WriteFile(filename, []byte(msg), 0o666)
	}
	os.Exit(exitCode)
}

// XXX: reimplement make_url?  Or can we rely on DET_MASTER from the master being correct?
