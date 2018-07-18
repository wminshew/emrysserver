DATE := $(shell date +%Y-%m-%d_%H-%M-%S)

build: cloudbuild.yaml emrysserver.yaml
	gcloud container builds submit --config cloudbuild.yaml --substitutions=_IMAGE=emrysserver,_BUILD=$(DATE) .

deploy:
	kubectl set image deployment/emrys-deployment emrys-container=gcr.io/emrys-12/emrysserver:latest --record
