run:
	go run ./cmd/minecraft-bedrock-wrapper

run-docker:
	docker build -t minecraft-bedrock .
	docker run -it --rm -p 8080:8080 -p 19132:19132/udp -e EULA_ACCEPT=true -e AUTH_KEY=supersecret minecraft-bedrock