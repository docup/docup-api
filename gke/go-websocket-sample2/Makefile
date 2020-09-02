.PHONY: deploy
deploy:
	docker build -t gcr.io/docup-269111/go-websocket-sample:latest .
	docker push gcr.io/docup-269111/go-websocket-sample:latest
	kubectl set image deployment/wss wss=gcr.io/docup-269111/go-websocket-sample:latest
