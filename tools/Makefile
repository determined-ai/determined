.PHONY: start-db
start-db: stop-db
	docker-compose up -d

.PHONY: stop-db
stop-db:
	docker-compose down

.PHONY: run
run:
	@rm -rf build
	@mkdir -p build/webui/docs
	@mkdir -p build/webui/elm
	@mkdir -p build/webui/react
	@mkdir -p build/wheels/
	@ln -s $(abspath ../master/static) build/
	@ln -s $(abspath ../docs/site/html/*) build/webui/docs
	@ln -s $(abspath ../webui/react/build/*) build/webui/react
	@ln -s $(abspath ../webui/elm/public) build/webui/elm
	@ln -s $(abspath ../common/dist/*.whl) build/wheels/
	@ln -s $(abspath ../cli/dist/*.whl) build/wheels/
	@ln -s $(abspath ../harness/dist/*.whl) build/wheels/
	@python3 run-server.py
