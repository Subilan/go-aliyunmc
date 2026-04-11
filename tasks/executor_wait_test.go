package tasks

import (
	"testing"

	"go-aliyunmc/store/models"
)

func TestExecutorWaitSuccessReturnsNil(t *testing.T) {
	e := NewExecutor(&TaskDefinition{})
	e.task = &models.Task{Status: models.TaskStatusSuccess}

	waitCh := e.Wait()
	e.cancel()

	err := <-waitCh
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestExecutorWaitInterrupted(t *testing.T) {
	e := NewExecutor(&TaskDefinition{})
	e.task = &models.Task{Status: models.TaskStatusFailed, Error: "__INTERRUPTED__"}

	waitCh := e.Wait()
	e.cancel()

	err := <-waitCh
	if err == nil || err.Error() != "__INTERRUPTED__" {
		t.Fatalf("expected __INTERRUPTED__, got %v", err)
	}
}

func TestExecutorWaitTimeout(t *testing.T) {
	e := NewExecutor(&TaskDefinition{})
	e.task = &models.Task{Status: models.TaskStatusFailed, Error: "__TIMEOUT__"}

	waitCh := e.Wait()
	e.cancel()

	err := <-waitCh
	if err == nil || err.Error() != "__TIMEOUT__" {
		t.Fatalf("expected __TIMEOUT__, got %v", err)
	}
}

func TestExecutorWaitReturnsTaskErrorMessage(t *testing.T) {
	e := NewExecutor(&TaskDefinition{})
	e.task = &models.Task{Status: models.TaskStatusFailed, Error: "custom failure"}

	waitCh := e.Wait()
	e.cancel()

	err := <-waitCh
	if err == nil || err.Error() != "custom failure" {
		t.Fatalf("expected custom failure, got %v", err)
	}
}
