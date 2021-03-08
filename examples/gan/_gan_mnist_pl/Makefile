# TODO: remove this file before merging

dev:
	rm -f ./determined_common-0.14.4.dev0-py3-none-any.whl
	make -C ${PROJECT_ROOT}/common/ build
	cp ${PROJECT_ROOT}/common/dist/determined_common-0.14.4.dev0-py3-none-any.whl .

	rm -f ./determined-0.14.4.dev0-py3-none-any.whl
	make -C ${PROJECT_ROOT}/harness/ build
	cp ${PROJECT_ROOT}/harness/dist/determined-0.14.4.dev0-py3-none-any.whl .
