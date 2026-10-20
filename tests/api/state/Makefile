HURL=hurl --variables-file ../env/dev.env

snapshot-server-status:
	$(HURL) ./snapshot-server-status.hurl

snapshot-instance-status:
	$(HURL) ./snapshot-instance-status.hurl

watch-server-status:
	bash ./run_watch.sh /state/watch/server-status

watch-instance-status:
	bash ./run_watch.sh /state/watch/instance-status

watch-route:
	bash ./run_watch.sh $(path)
