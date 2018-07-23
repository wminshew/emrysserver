DATE := $(shell date +%Y-%m-%d_%H-%M-%S)

all: build deploy

user: build-user deploy-user

miner: build-miner deploy-miner

job: build-job deploy-job


build: cloudbuild.yaml
	gcloud container builds submit --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-user: cmd/user/cloudbuild.yaml cmd/user/dockerfile
	gcloud container builds submit --config ./cmd/user/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-miner: cmd/miner/cloudbuild.yaml cmd/miner/dockerfile
	gcloud container builds submit --config ./cmd/miner/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-job: cmd/job/cloudbuild.yaml cmd/job/dockerfile
	gcloud container builds submit --config ./cmd/job/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .


deploy: deploy-user deploy-miner deploy-job deploy-sqlproxy deploy-ing

deploy-user: cmd/user/svc-deploy.yaml
	kubectl apply -f cmd/user/svc-deploy.yaml

deploy-miner: cmd/miner/svc-deploy.yaml
	kubectl apply -f cmd/miner/svc-deploy.yaml

deploy-job: cmd/job/svc-deploy.yaml
	kubectl apply -f cmd/job/svc-deploy.yaml

deploy-sqlproxy: cmd/sqlproxy/svc-deploy.yaml

deploy-ing: ingress.yaml
	kubectl apply -f ingress.yaml


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
