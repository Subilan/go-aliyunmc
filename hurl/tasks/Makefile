# hurl/tasks/Makefile
HURL=hurl --variables-file ../env/dev.env

.PHONY: test-create-test-task test-sse

test-create-test-task:
	$(HURL) create-test-task.hurl --variable username=$(USER)

test-sse:
	$(HURL) sse.hurl --variable username=$(USER) --variable taskId=$(TASK)