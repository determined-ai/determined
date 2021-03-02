.PHONY: fmt check test

fmt:
	isort \
		-s ./tests/fixtures/pytorch_amp/model_def.py \
		-s ./tests/fixtures/pytorch_amp/layers.py \
		-s ./tests/fixtures/pytorch_amp/data.py
	# These skipped files are symlinks to another module
	black .

check:
	isort --check-only  \
		-s ./tests/fixtures/pytorch_amp/model_def.py \
		-s ./tests/fixtures/pytorch_amp/layers.py \
		-s ./tests/fixtures/pytorch_amp/data.py
	black . --check
	flake8
	mypy tests

test:
	pytest -vv -s --durations=0 tests
