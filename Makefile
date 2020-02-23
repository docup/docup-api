.PHONY: appengine/dev/deploy
appengine/dev/deploy:
	gcloud app deploy --project=soilworks-expt-01-266813 appengine.yaml --quiet

.PHONY: cloudtask/dev/create-task
cloudtask/dev/create-task:	// for debug. enqueue dummy massage to 'default' queue
	gcloud tasks create-http-task --queue=default --project=soilworks-expt-01-266813 --oidc-service-account-email=soilworks-expt-01-266813@appspot.gserviceaccount.com --url=https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app/cloudtasks --oidc-token-audience=https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app
