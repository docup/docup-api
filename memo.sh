# The deployment step is describe here https://cloud.google.com/endpoints/docs/openapi/get-started-app-engine-standard?hl=ja#configure_iap

gcloud config set run/region asia-northeast1

gcloud run deploy endpoints-runtime-serverless \
    --image="gcr.io/endpoints-release/endpoints-runtime-serverless:1.30.0" \
    --allow-unauthenticated

gcloud endpoints services deploy openapi-appengine.yaml

#gcloud run services update endpoints-runtime-serverless \
#   --set-env-vars ENDPOINTS_SERVICE_NAME=endpoints-runtime-serverless-g7vn673b6q-an.a.run.app

gcloud run services update endpoints-runtime-serverless \
   --set-env-vars="^|^ENDPOINTS_SERVICE_NAME=endpoints-runtime-serverless-g7vn673b6q-an.a.run.app|ESP_ARGS=--rollout_strategy=managed,--cors_preset=basic"

curl --request GET \
   --header "content-type:application/json" \
   "https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app/api"

curl -H "Authorization: Bearer ${TOKEN}" "https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app/api"

curl -H 'Content-Type:application/json' -d "{}" "https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app/cloudtasks"

curl -H 'Content-Type:application/json' -d "{}" "https://api-dot-soilworks-expt-01-266813.appspot.com/cloudtasks"

gcloud tasks create-http-task --queue=default --project=soilworks-expt-01-266813 --oidc-service-account-email=soilworks-expt-01-266813@appspot.gserviceaccount.com --url=https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app/cloudtasks --oidc-token-audience=https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app