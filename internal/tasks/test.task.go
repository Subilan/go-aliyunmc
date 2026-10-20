package tasks

import (
	"fmt"
	"time"
)

func testTask(tc *TaskContext, _ map[string]any) error {
	for i := 0; i < 30; i++ {
		if i%10 == 0 {
			tc.nextStep()
		}
		tc.println(fmt.Sprintf("test step %d", i))
		time.Sleep(time.Second)
	}
	tc.done()
	return nil
}
