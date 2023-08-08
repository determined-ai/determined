.PHONY: clean
clean:
	rm -rf build/

.PHONY: prep-root
prep-root:
	@rm -rf build
	@mkdir -p build/webui/docs
	@mkdir -p build/webui/react
	@mkdir -p build/wheels/
	@ln -s $(abspath ../master/static) build/
	@ln -s $(abspath ../proto/build/swagger) build/swagger
	@ln -s $(abspath ../docs/site/html/*) build/webui/docs
	@ln -s $(abspath ../webui/react/build/*) build/webui/react
	@ln -s $(abspath ../harness/dist/*.whl) build/wheels/

.PHONY: fmt
fmt:
	isort .
	black .

.PHONY: check
check:
	isort . --check-only
	black . --check
	python -m flake8
	mypy scripts
