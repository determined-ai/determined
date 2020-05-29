.PHONY: clean
clean:
	rm -rf build/

.PHONY: start-db
start-db: stop-db
	docker-compose up -d

.PHONY: stop-db
stop-db:
	docker-compose down

.PHONY: psql
psql:
	docker exec -it \
		tools_determined-db_1 \
		psql determined -U postgres

.PHONY: prep-root
prep-root:
	@rm -rf build
	@mkdir -p build/webui/docs
	@mkdir -p build/webui/elm
	@mkdir -p build/webui/react
	@mkdir -p build/wheels/
	@ln -s $(abspath $(PROJECT_ROOT)/master/static) build/
	@ln -s $(abspath $(PROJECT_ROOT)/docs/site/html/*) build/webui/docs
	@ln -s $(abspath $(PROJECT_ROOT)/webui/react/build/*) build/webui/react
	@ln -s $(abspath $(PROJECT_ROOT)/webui/elm/public) build/webui/elm
	@ln -s $(abspath $(PROJECT_ROOT)/common/dist/*.whl) build/wheels/
	@ln -s $(abspath $(PROJECT_ROOT)/cli/dist/*.whl) build/wheels/
	@ln -s $(abspath $(PROJECT_ROOT)/harness/dist/*.whl) build/wheels/

.PHONY: run
run: prep-root
	@python3 run-server.py
