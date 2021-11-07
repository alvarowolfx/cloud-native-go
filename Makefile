dockerize:
	docker build --file cmd/api/Dockerfile -t cngo-api .
	docker build --file cmd/worker/Dockerfile -t cngo-worker .