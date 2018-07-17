build: cloudbuild.yaml emrysserver.yaml
	gcloud container builds submit --config cloudbuild.yaml --substitutions=_IMAGE=emrysserver,_BUILD=$(date +%Y-%m-%d_%H-%M-%S) .

deploy:
	kubectl set image deployment/emrys-deployment emrys-container=gcr.io/emrys-12/emrysserver:latest
