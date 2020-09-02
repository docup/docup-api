.PHONY: deploy

COMMITHASH = $(shell git log --pretty=format:'%h' -n 1)
VERSION ?= commit-$(COMMITHASH)

deploy:
	docker build -t gcr.io/docup-269111/go-websocket-sample:$(VERSION) .
	docker push gcr.io/docup-269111/go-websocket-sample:$(VERSION)
	kubectl set image deployment/wss wss=gcr.io/docup-269111/go-websocket-sample:$(VERSION)
