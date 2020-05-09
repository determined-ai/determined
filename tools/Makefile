.PHONY: start-db
start-db: stop-db
	docker-compose up -d
	docker-compose logs -f

.PHONY: stop-db
stop-db:
	docker-compose down

.PHONY: run
run:
	./run-server.sh
