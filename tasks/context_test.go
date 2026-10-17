package tasks

import (
	"fmt"
	"testing"
	"time"
)

// 输出突发不应丢行：println 必须把缓冲区容量以内的每一行都交给 executor。
func TestTaskContextPrintlnKeepsBurst(t *testing.T) {
	tc := NewTaskContext()
	defer tc.done()

	const count = 500
	for i := 0; i < count; i++ {
		tc.println(fmt.Sprintf("line %d", i))
	}

	for i := 0; i < count; i++ {
		select {
		case output := <-tc.outputChan:
			want := fmt.Sprintf("line %d", i)
			if output.Output != want {
				t.Fatalf("output %d: got %q, want %q", i, output.Output, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("missing output %d", i)
		}
	}
}

// 缓冲区满时 println 应形成背压，并在任务取消后立即解除阻塞。
func TestTaskContextPrintlnUnblocksOnCancel(t *testing.T) {
	tc := NewTaskContext()

	for i := 0; i < taskOutputChanBuffer; i++ {
		tc.println("fill")
	}

	done := make(chan struct{})
	go func() {
		tc.println("blocked")
		close(done)
	}()

	tc.done()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("println remained blocked after cancel")
	}
}
