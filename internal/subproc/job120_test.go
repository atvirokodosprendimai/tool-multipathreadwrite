//go:build windows || mrwjobs

package subproc

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestJobHelper is not a test: it is the child the job tests start, chosen by
// MRW_JOB_HELPER. Run directly it returns at once.
func TestJobHelper(t *testing.T) {
	mode := os.Getenv("MRW_JOB_HELPER")
	if mode == "" {
		return
	}
	switch mode {
	case "exit0":
		os.Exit(0)
	case "sleep":
		time.Sleep(30 * time.Second)
	case "grandchild":
		_ = os.WriteFile(os.Getenv("MRW_JOB_PIDFILE"), []byte(strconv.Itoa(os.Getpid())), 0o644)
		time.Sleep(60 * time.Second)
	case "spawn-sleep", "spawn-exit":
		g := exec.Command(os.Args[0], "-test.run=^TestJobHelper$")
		g.Env = append(os.Environ(), "MRW_JOB_HELPER=grandchild")
		if err := g.Start(); err != nil {
			os.Exit(3)
		}
		for i := 0; i < 200; i++ {
			if _, err := os.Stat(os.Getenv("MRW_JOB_PIDFILE")); err == nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if mode == "spawn-sleep" {
			time.Sleep(30 * time.Second)
		}
	}
	os.Exit(0)
}

// helper is a child running TestJobHelper in mode.
func helper(mode string, env ...string) *exec.Cmd {
	c := exec.Command(os.Args[0], "-test.run=^TestJobHelper$")
	c.Env = append(append(os.Environ(), "MRW_JOB_HELPER="+mode), env...)
	return c
}

// fakeJobs records the calls runInJob makes, and fails the ones a test names.
type fakeJobs struct {
	mu        sync.Mutex
	calls     []string
	pid       int
	assignErr error
}

func (f *fakeJobs) record(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, s)
}

func (f *fakeJobs) create() (uintptr, error) { f.record("create"); return 7, nil }
func (f *fakeJobs) assign(job uintptr, pid int) error {
	f.record("assign")
	f.pid = pid
	return f.assignErr
}
func (f *fakeJobs) resume(pid int) error        { f.record("resume"); return nil }
func (f *fakeJobs) terminate(job uintptr) error { f.record("terminate"); return nil }
func (f *fakeJobs) close(job uintptr) error     { f.record("close"); return nil }

func (f *fakeJobs) got() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// ADR-120. The child is assigned to the job before its main thread is resumed,
// so nothing it starts can be created outside the job; the job is terminated
// and closed once the child exits, taking a grandchild left behind.
func TestTheJobSequenceAssignsBeforeResuming(t *testing.T) {
	f := &fakeJobs{}
	c := helper("exit0")
	if err := runInJob(c, &jobTree{api: f}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"create", "assign", "resume", "terminate", "close"}; !reflect.DeepEqual(f.got(), want) {
		t.Fatalf("calls %v, want %v", f.got(), want)
	}
	if f.pid != c.Process.Pid {
		t.Fatalf("assigned pid %d, the child is %d", f.pid, c.Process.Pid)
	}
}

// A child that cannot be contained does not run uncontained: it is killed,
// waited for, its job closed, and the error returned.
func TestAFailedAssignKillsTheChildAndClosesTheJob(t *testing.T) {
	f := &fakeJobs{assignErr: errors.New("access denied")}
	c := helper("sleep")
	start := time.Now()
	err := runInJob(c, &jobTree{api: f})
	if !errors.Is(err, ErrNotContained) || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("err %v, want ErrNotContained wrapping the assign error", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatalf("took %v: the child was not killed", time.Since(start))
	}
	// Killed while suspended, it ran nothing: no exit status is reported, so the
	// check says it could not run rather than that the check failed.
	if c.ProcessState != nil {
		t.Fatalf("ProcessState %v: a child that never ran reports an exit status", c.ProcessState)
	}
	if want := []string{"create", "assign", "close"}; !reflect.DeepEqual(f.got(), want) {
		t.Fatalf("calls %v, want %v", f.got(), want)
	}
}

// A cancel that lands while the child is being contained is reported as the
// cancel, so the check says interrupted, not that the child could not start.
func TestACancelDuringContainmentIsReportedAsTheCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &fakeJobs{assignErr: errors.New("the process has exited")}
	err := runInJob(helper("sleep"), &jobTree{api: f, ctx: ctx})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrNotContained) {
		t.Fatalf("err %v, want context.Canceled and ErrNotContained", err)
	}
}

// A cancel before the child is in its job kills the child alone; once it is
// assigned, the cancel terminates the job, which takes every member.
func TestACancelTerminatesTheJobOnceAssigned(t *testing.T) {
	f := &fakeJobs{}
	tree := &jobTree{api: f}
	kills := 0
	kill := func() error { kills++; return nil }
	_ = tree.cancel(kill)
	if kills != 1 || len(f.got()) != 0 {
		t.Fatalf("before assignment: kills %d, calls %v; want the child killed", kills, f.got())
	}
	tree.job = 7
	_ = tree.cancel(kill)
	if kills != 1 || !reflect.DeepEqual(f.got(), []string{"terminate"}) {
		t.Fatalf("after assignment: kills %d, calls %v; want the job terminated", kills, f.got())
	}
}

// Git for Windows' sh breaks away from any job that allows it (a peer's
// measurement, 2026-10-02): the job kills on close and allows nothing else.
func TestTheJobLimitsKillOnCloseAndNothingElse(t *testing.T) {
	if got := killOnCloseLimits().BasicLimitInformation.LimitFlags; got != jobObjectLimitKillOnJobClose {
		t.Fatalf("LimitFlags %#x, want %#x (KILL_ON_JOB_CLOSE alone)", got, jobObjectLimitKillOnJobClose)
	}
}

// The struct SetInformationJobObject reads is declared by hand: on 64-bit
// Windows JOBOBJECT_EXTENDED_LIMIT_INFORMATION is 144 bytes.
func TestTheJobLimitStructHasTheWin32Size(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the size pinned is the 64-bit one")
	}
	if got := unsafe.Sizeof(extendedLimitInformation{}); got != 144 {
		t.Fatalf("extendedLimitInformation is %d bytes, Win32 says 144", got)
	}
}
