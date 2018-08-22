DATE := $(shell date +%Y-%m-%d_%H-%M-%S)
MINERTIMEOUT := 605
JOBTIMEOUT := 125
IMAGETIMEOUT := 305
DATATIMEOUT := 305

all: build deploy rollout

user: build-user deploy-user rollout-user

miner: build-miner deploy-miner rollout-miner

job: build-job deploy-job rollout-job

image: build-image deploy-image

data: build-data deploy-data rollout-data

devpi: build-devpi deploy-devpi


build: cloudbuild.yaml dep-ensure
	# container-builder-local --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

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

build-image: cmd/image/cloudbuild.yaml cmd/image/dockerfile dep-ensure
	# container-builder-local --config ./cmd/image/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/image/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/image/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-data: cmd/data/cloudbuild.yaml cmd/data/dockerfile dep-ensure
	# container-builder-local --config ./cmd/data/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/data/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/data/cloudbuild.yaml --substitutions=_BUILD=$(DATE) .

build-devpi: cmd/devpi/cloudbuild.yaml cmd/devpi/dockerfile dep-ensure
	# container-builder-local --config ./cmd/devpi/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=true --push=false .
	# container-builder-local --config ./cmd/devpi/cloudbuild.yaml --substitutions=_BUILD=$(DATE) --dryrun=false --push=false .
	gcloud builds submit --config ./cmd/devpi/cloudbuild.yaml --substitutions=_BUILD=$(DATE) ./cmd/devpi/

dep-ensure:
	dep ensure -v

deploy: deploy-user deploy-miner deploy-job deploy-image deploy-data deploy-sqlproxy deploy-devpi deploy-ing

deploy-user: cmd/user/svc-deploy.yaml
	kubectl apply -f cmd/user/svc-deploy.yaml

deploy-miner: cmd/miner/svc-deploy.yaml
	kubectl apply -f cmd/miner/svc-deploy.yaml
	gcloud compute backend-services list --filter='miner' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(MINERTIMEOUT)

deploy-job: cmd/job/svc-sts.yaml
	kubectl apply -f cmd/job/svc-sts.yaml
	gcloud compute backend-services list --filter='job' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(JOBTIMEOUT)

deploy-image: cmd/image/svc-deploy.yaml
	kubectl create configmap registry-config --dry-run -o yaml --from-file=cmd/image/registry-config.yaml | kubectl apply -f -
	kubectl apply -f cmd/image/svc-deploy.yaml
	gcloud compute backend-services list --filter='image' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(IMAGETIMEOUT)

deploy-data: cmd/data/svc-sts.yaml
	kubectl apply -f cmd/data/svc-sts.yaml
	gcloud compute backend-services list --filter='data' --format='value(name)' | xargs -n 1 gcloud compute backend-services update --global --timeout $(DATATIMEOUT)

deploy-sqlproxy: cmd/sqlproxy/svc-deploy.yaml
	kubectl apply -f cmd/sqlproxy/svc-deploy.yaml

deploy-devpi: cmd/devpi/svc-sts.yaml
	kubectl apply -f cmd/devpi/svc-sts.yaml

deploy-ing: emrys-ing.yaml
	kubectl replace -f emrys-ing.yaml


rollout: rollout-user rollout-miner rollout-job rollout-image rollout-data

rollout-user:
	kubectl set image deploy/user-deploy user-container=gcr.io/emrys-12/user:latest
	kubectl rollout status deploy/user-deploy

rollout-miner:
	kubectl set image deploy/miner-deploy miner-container=gcr.io/emrys-12/miner:latest
	kubectl rollout status deploy/miner-deploy

rollout-job:
	kubectl set image sts/job-sts job-container=gcr.io/emrys-12/job:latest
	kubectl rollout status sts/job-sts

rollout-image:
	kubectl set image deploy/image-deploy image-container=gcr.io/emrys-12/image:latest
	kubectl rollout status deploy/image-deploy

rollout-data:
	kubectl set image sts/data-sts data-container=gcr.io/emrys-12/data:latest
	kubectl rollout status sts/data-sts

rollout-devpi:
	kubectl set image sts/devpi-sts devpi-container=gcr.io/emrys-12/devpi:latest
	kubectl rollout status sts/devpi-sts


rollback: rollback-user rollback-miner rollback-job rollback-image rollback-data

rollback-user:
	kubectl rollout undo deploy/user-deploy
	kubectl rollout status deploy/user-deploy

rollback-miner:
	kubectl rollout undo deploy/miner-deploy
	kubectl rollout status deploy/miner-deploy

rollback-job:
	kubectl rollout undo sts/job-sts
	kubectl rollout status sts/job-sts

rollback-image:
	kubectl rollout undo deploy/image-deploy
	kubectl rollout status deploy/image-deploy

rollback-data:
	kubectl rollout undo deploy/data-deploy
	kubectl rollout status deploy/data-deploy

rollback-devpi:
	kubectl rollout undo sts/devpi-sts
	kubectl rollout status sts/devpi-sts
