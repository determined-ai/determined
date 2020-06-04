 .PHONY: get-deps
get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

public/styles.css: styles-in.css tailwind.config.js
	npm run build-css -- -o public/styles.css

.PHONY: build
build: public/styles.css
	npm run build-elm -- --output=public/determined-ui.js

.PHONY: clean
clean:
	rm -f npm-debug.log
	rm -f public/styles.css
	rm -f public/determined-ui.js
	rm -rf elm-stuff/ tests/elm-stuff/ node_modules/

.PHONY: live
live:
	npm run live-workaround -- --output=public/determined-ui.js

.PHONY: fmt
fmt:
	npm run format -- --yes src/ tests/

.PHONY: check
check:
	npm run format -- --validate src/ tests/
	npm run analyse

.PHONY: test
test:
	npm run test
