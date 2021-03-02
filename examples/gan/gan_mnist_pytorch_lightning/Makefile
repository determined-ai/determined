# TODO: remove this file before merging

dev:
	rm -f ./determined_common-0.14.3.dev0-py3-none-any.whl
	make -C ~/determined/common/ build
	cp ~/determined/common/dist/determined_common-0.14.3.dev0-py3-none-any.whl .

	rm -f ./determined-0.14.3.dev0-py3-none-any.whl
	make -C ~/determined/harness/ build
	cp ~/determined/harness/dist/determined-0.14.3.dev0-py3-none-any.whl .
