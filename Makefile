DATE := $(shell date +%Y-%m-%d_%H-%M-%S)
USERTIMEOUT := 120
MINERTIMEOUT := 605

all: build deploy rollout

user: build-user deploy-user rollout-user

miner: build-miner deploy-miner rollout-miner

job: build-job deploy-job rollout-job


build: cloudbuild.yaml
	gcloud container builds submit --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-user: cmd/user/cloudbuild.yaml cmd/user/dockerfile
	gcloud container builds submit --config ./cmd/user/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-miner: cmd/miner/cloudbuild.yaml cmd/miner/dockerfile
	gcloud container builds submit --config ./cmd/miner/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-job: cmd/job/cloudbuild.yaml cmd/job/dockerfile
	gcloud container builds submit --config ./cmd/job/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .


deploy: deploy-user deploy-miner deploy-job deploy-docker deploy-sqlproxy deploy-ing

deploy-user: cmd/user/svc-deploy.yaml
	kubectl apply -f cmd/user/svc-deploy.yaml
	gcloud compute backend-services list --filter='user' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(USERTIMEOUT)

deploy-miner: cmd/miner/svc-deploy.yaml
	kubectl apply -f cmd/miner/svc-deploy.yaml
	gcloud compute backend-services list --filter='miner' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(MINERTIMEOUT)

deploy-job: cmd/job/svc-deploy.yaml
	kubectl apply -f cmd/job/svc-deploy.yaml

deploy-docker: cmd/docker/svc-deploy.yaml
	kubectl apply -f cmd/docker/svc-deploy.yaml

deploy-sqlproxy: cmd/sqlproxy/svc-deploy.yaml
	kubectl apply -f cmd/sqlproxy/svc-deploy.yaml

deploy-ing: gce-ing.yaml
	kubectl replace -f gce-ing.yaml


rollout: rollout-user rollout-miner rollout-job

rollout-user:
	kubectl set image deploy/user-deploy user-container=gcr.io/emrys-12/user:latest
	kubectl rollout status deploy/user-deploy

rollout-miner:
	kubectl set image deploy/miner-deploy miner-container=gcr.io/emrys-12/miner:latest
	kubectl rollout status deploy/miner-deploy

rollout-job:
	kubectl set image deploy/job-deploy job-container=gcr.io/emrys-12/job:latest
	kubectl rollout status deploy/job-deploy


rollback: rollback-user rollback-miner rollback-job

rollback-user:
	kubectl rollout undo deploy/user-deploy
	kubectl rollout status deploy/user-deploy

rollback-miner:
	kubectl rollout undo deploy/miner-deploy
	kubectl rollout status deploy/miner-deploy

rollback-job:
	kubectl rollout undo deploy/job-deploy
	kubectl rollout status deploy/job-deploy
