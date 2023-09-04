up:
	docker-compose -f docker/docker-compose.yaml -p binlogtest up -d

logs:
	docker-compose -f docker/docker-compose.yaml -p binlogtest logs -f

down:
	docker-compose -f docker/docker-compose.yaml -p binlogtest down


run:
	go run binlogtest