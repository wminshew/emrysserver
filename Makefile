DATE := $(shell date +%Y-%m-%d_%H-%M-%S)
MINER_TIMEOUT := 620
JOB_TIMEOUT := 140
IMAGE_TIMEOUT := 320
REGISTRY_TIMEOUT := 320
DATA_TIMEOUT := 320
MAX_RPS_PER_INSTANCE := 100
# DEVPITIMEOUT := 320

all: build deploy

default-backend: build-default-backend deploy-default-backend

ing: deploy-ing

auth: build-auth deploy-auth

user: build-user deploy-user

miner: build-miner deploy-miner

job: build-job deploy-job

notebook: build-notebook deploy-notebook

image: build-image deploy-image

registry: build-registry deploy-registry

data: build-data deploy-data

devpi: build-devpi deploy-devpi

dep-ensure:
	dep ensure -v

build: cloudbuild.yaml dep-ensure
	# container-builder-local --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-default-backend: cmd/default-backend/cloudbuild.yaml cmd/default-backend/dockerfile dep-ensure
	# container-builder-local --config ./cmd/default-backend/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/default-backend/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/default-backend/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-auth: cmd/auth/cloudbuild.yaml cmd/auth/dockerfile dep-ensure
	# container-builder-local --config ./cmd/auth/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/auth/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/auth/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-user: cmd/user/cloudbuild.yaml cmd/user/dockerfile dep-ensure
	# container-builder-local --config ./cmd/user/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/user/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/user/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-miner: cmd/miner/cloudbuild.yaml cmd/miner/dockerfile dep-ensure
	# container-builder-local --config ./cmd/miner/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/miner/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/miner/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-job: cmd/job/cloudbuild.yaml cmd/job/dockerfile dep-ensure
	# container-builder-local --config ./cmd/job/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/job/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/job/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-notebook: cmd/notebook/cloudbuild.yaml cmd/notebook/dockerfile cmd/notebook/dockerfile-sshd cmd/notebook/entrypoint-sshd.sh dep-ensure
	# container-builder-local --config ./cmd/notebook/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/notebook/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/notebook/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-image: cmd/image/cloudbuild.yaml cmd/image/dockerfile dep-ensure
	# container-builder-local --config ./cmd/image/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/image/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/image/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-registry: cmd/registry/cloudbuild.yaml cmd/registry/dockerfile dep-ensure
	# container-builder-local --config ./cmd/registry/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/registry/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/registry/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-data: cmd/data/cloudbuild.yaml cmd/data/dockerfile dep-ensure
	# container-builder-local --config ./cmd/data/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/data/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/data/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-devpi: cmd/devpi/cloudbuild.yaml cmd/devpi/dockerfile dep-ensure
	# container-builder-local --config ./cmd/devpi/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/devpi/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/devpi/cloudbuild.yaml --substitutions=_BUILD=$(DATE) ./cmd/devpi/


deploy: deploy-default-backend deploy-auth deploy-user deploy-miner deploy-job deploy-notebook deploy-image deploy-registry deploy-data deploy-sqlproxy deploy-devpi deploy-ing

deploy-default-backend: cmd/default-backend/svc-deploy.yaml
	kubectl apply -f cmd/default-backend/svc-deploy.yaml

deploy-auth: cmd/auth/svc-deploy.yaml
	kubectl apply -f cmd/auth/svc-deploy.yaml
	gcloud compute backend-services list --filter='auth' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-user: cmd/user/svc-deploy.yaml
	kubectl apply -f cmd/user/svc-deploy.yaml
	gcloud compute backend-services list --filter='user' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-miner: cmd/miner/svc-deploy.yaml
	kubectl apply -f cmd/miner/svc-deploy.yaml
	gcloud compute backend-services list --filter='miner' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(MINER_TIMEOUT)
	gcloud compute backend-services list --filter='miner' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a


deploy-job: cmd/job/svc-sts.yaml
	kubectl apply -f cmd/job/svc-sts.yaml
	gcloud compute backend-services list --filter='job' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(JOB_TIMEOUT)
	gcloud compute backend-services list --filter='job' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-notebook: cmd/notebook/svc-deploy.yaml
	kubectl apply -f cmd/notebook/svc-deploy.yaml
	# gcloud compute backend-services list --filter='notebook' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-image: cmd/image/svc-deploy.yaml
	kubectl create configmap image-registry-config --dry-run -o yaml --from-file=cmd/image/registry-config.yaml | kubectl apply -f -
	kubectl apply -f cmd/image/svc-deploy.yaml
	gcloud compute backend-services list --filter='image' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(IMAGE_TIMEOUT)
	gcloud compute backend-services list --filter='image' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-registry: cmd/registry/svc-deploy.yaml
	kubectl create configmap registry-registry-config --dry-run -o yaml --from-file=cmd/registry/registry-config.yaml | kubectl apply -f -
	kubectl apply -f cmd/registry/svc-deploy.yaml
	gcloud compute backend-services list --filter='registry' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(REGISTRY_TIMEOUT)
	gcloud compute backend-services list --filter='registry' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-data: cmd/data/svc-sts.yaml
	kubectl apply -f cmd/data/svc-sts.yaml
	gcloud compute backend-services list --filter='data' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(DATA_TIMEOUT)
	gcloud compute backend-services list --filter='data' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-sqlproxy: cmd/sqlproxy/svc-deploy.yaml
	kubectl apply -f cmd/sqlproxy/svc-deploy.yaml
	# gcloud compute backend-services list --filter='sqlproxy' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-devpi: cmd/devpi/svc-sts.yaml
	kubectl apply -f cmd/devpi/svc-sts.yaml
	# gcloud compute backend-services list --filter='devpi' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(DEVPITIMEOUT)
	# gcloud compute backend-services list --filter='devpi' --format='value(name)' | xargs -n 1 gcloud compute backend-services update-backend --max-rate-per-instance $(MAX_RPS_PER_INSTANCE) --global --instance-group=k8s-ig--5e862efea9931d79 --instance-group-zone=us-central1-a

deploy-ing: emrys-ing.yaml
	kubectl replace -f emrys-ing.yaml


rollout: rollout-default-backend rollout-auth rollout-user rollout-miner rollout-job rollout-image rollout-registry rollout-data

rollout-default-backend:
	kubectl set image deploy/default-backend-deploy default-backend-container=gcr.io/emrys-12/default-backend:latest
	kubectl rollout status deploy/default-backend-deploy

rollout-auth:
	kubectl set image deploy/auth-deploy auth-container=gcr.io/emrys-12/auth:latest
	kubectl rollout status deploy/auth-deploy

rollout-user:
	kubectl set image deploy/user-deploy user-container=gcr.io/emrys-12/user:latest
	kubectl rollout status deploy/user-deploy

rollout-miner:
	kubectl set image deploy/miner-deploy miner-container=gcr.io/emrys-12/miner:latest
	kubectl rollout status deploy/miner-deploy

rollout-job:
	kubectl set image sts/job-sts job-container=gcr.io/emrys-12/job:latest
	kubectl rollout status sts/job-sts

rollout-notebook:
	kubectl set image deploy/notebook-deploy notebook-container=gcr.io/emrys-12/notebook:latest
	kubectl set image deploy/notebook-deploy notebook-sshd-container=gcr.io/emrys-12/notebook-sshd:latest
	kubectl rollout status deploy/notebook-deploy

rollout-image:
	kubectl set image deploy/image-deploy image-container=gcr.io/emrys-12/image:latest
	kubectl rollout status deploy/image-deploy

rollout-registry:
	kubectl set image deploy/registry-deploy image-container=gcr.io/emrys-12/registry:latest
	kubectl rollout status deploy/registry-deploy

rollout-data:
	kubectl set image sts/data-sts data-container=gcr.io/emrys-12/data:latest
	kubectl rollout status sts/data-sts

rollout-devpi:
	kubectl set image sts/devpi-sts devpi-container=gcr.io/emrys-12/devpi:latest
	kubectl rollout status sts/devpi-sts


rollback: rollback-default-backend rollback-auth rollback-user rollback-miner rollback-job rollback-notebook rollback-image rollback-registry rollback-data

rollback-default-backend:
	kubectl rollout undo deploy/default-backend-deploy
	kubectl rollout status deploy/default-backend-deploy

rollback-auth:
	kubectl rollout undo deploy/auth-deploy
	kubectl rollout status deploy/auth-deploy

rollback-user:
	kubectl rollout undo deploy/user-deploy
	kubectl rollout status deploy/user-deploy

rollback-miner:
	kubectl rollout undo deploy/miner-deploy
	kubectl rollout status deploy/miner-deploy

rollback-job:
	kubectl rollout undo sts/job-sts
	kubectl rollout status sts/job-sts

rollback-notebook:
	kubectl rollout undo deploy/notebook-deploy
	kubectl rollout status deploy/notebook-deploy

rollback-image:
	kubectl rollout undo deploy/image-deploy
	kubectl rollout status deploy/image-deploy

rollback-registry:
	kubectl rollout undo deploy/registry-deploy
	kubectl rollout status deploy/registry-deploy

rollback-data:
	kubectl rollout undo deploy/data-deploy
	kubectl rollout status deploy/data-deploy

rollback-devpi:
	kubectl rollout undo sts/devpi-sts
	kubectl rollout status sts/devpi-sts


delete: delete-default-backend delete-auth delete-user delete-miner delete-job delete-notebook delete-image delete-registry delete-data

delete-default-backend:
	kubectl delete pod -lapp=default-backend

delete-auth:
	kubectl delete pod -lapp=auth

delete-user:
	kubectl delete pod -lapp=user

delete-miner:
	kubectl delete pod -lapp=miner

delete-job:
	kubectl delete pod -lapp=job

delete-notebook:
	kubectl delete pod -lapp=notebook

delete-image:
	kubectl delete pod -lapp=image

delete-registry:
	kubectl delete pod -lapp=registry

delete-data:
	kubectl delete pod -lapp=data

delete-devpi:
	kubectl delete pod -lapp=devpi
