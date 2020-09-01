fmt:
	helm lint charts/determined/

.PHONY: build
build: clean
	mkdir -p build/
	cp -r charts/determined build/determined-helm-chart
	tar -C build -czf build/determined-helm-chart.tgz determined-helm-chart
	rm -rf build/determined-helm-chart

.PHONY: clean
clean:
	rm -rf build/
