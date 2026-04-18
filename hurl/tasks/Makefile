# hurl/tasks/Makefile
HURL=hurl --variables-file ../env/dev.env
instance?=ecs.e-c1m2.large
zone?=cn-shenzhen-f
taskType?=create_instance

backup:
	bash ./run_and_watch.sh ./backup.hurl

archive:
	bash ./run_and_watch.sh ./archive.hurl

create-instance:
	$(HURL) ./create-instance.hurl --variable instance=$(instance) --variable zone=$(zone)

watch-hurl:
	bash ./run_and_watch.sh $(file) $(opts)

create-instance-watch:
	bash ./run_and_watch.sh ./create-instance.hurl --variable instance=$(instance) --variable zone=$(zone)

deploy-instance-watch:
	bash ./run_and_watch.sh ./deploy-instance.hurl
	
start-server-watch:
	bash ./run_and_watch.sh ./start-server.hurl

get-def:
	$(HURL) ./definition.hurl --variable taskType=$(taskType)