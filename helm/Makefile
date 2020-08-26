fmt:
	helm lint charts/determined/

.PHONY: build
build:
	mkdir -p build/
	tar -czf build/determined-helm-chart.tgz charts/determined

.PHONY: clean
clean:
	rm -rf build/
