DATE := $(shell date +%Y-%m-%d_%H-%M-%S)

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

build-notebook: cmd/notebook/cloudbuild.yaml cmd/notebook/dockerfile cmd/notebook/entrypoint.sh dep-ensure
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
	linkerd inject cmd/default-backend/svc-deploy.yaml | kubectl apply -f -

deploy-auth: cmd/auth/svc-deploy.yaml
	linkerd inject cmd/auth/svc-deploy.yaml | kubectl apply -f -

deploy-user: cmd/user/svc-deploy.yaml
	linkerd inject cmd/user/svc-deploy.yaml | kubectl apply -f -

deploy-miner: cmd/miner/svc-deploy.yaml
	linkerd inject cmd/miner/svc-deploy.yaml | kubectl apply -f -

deploy-job: cmd/job/svc-sts.yaml
	linkerd inject cmd/job/svc-sts.yaml | kubectl apply -f -

deploy-notebook: cmd/notebook/svc-deploy.yaml
	linkerd inject cmd/notebook/svc-deploy.yaml | kubectl apply -f -

deploy-image: cmd/image/svc-deploy.yaml
	kubectl apply -f cmd/image/svc-deploy.yaml
	# linkerd inject cmd/image/svc-deploy.yaml | kubectl apply -f -

deploy-registry: cmd/registry/svc-deploy.yaml
	linkerd inject cmd/registry/svc-deploy.yaml | kubectl apply -f -

deploy-data: cmd/data/svc-sts.yaml
	linkerd inject cmd/data/svc-sts.yaml | kubectl apply -f -

deploy-sqlproxy: cmd/sqlproxy/svc-deploy.yaml
	linkerd inject cmd/sqlproxy/svc-deploy.yaml | kubectl apply -f -

deploy-devpi: cmd/devpi/svc-sts.yaml
	linkerd inject cmd/devpi/svc-sts.yaml | kubectl apply -f -

deploy-ing: emrys-ing.yaml
	kubectl replace -f emrys-ing.yaml


rollout: rollout-default-backend rollout-auth rollout-user rollout-miner rollout-job rollout-image rollout-registry rollout-data

rollout-default-backend:
	kubectl set image deploy/default-backend-deploy default-backend-container=gcr.io/emrys-12/default-backend:latest -n emrys-prod
	kubectl rollout status deploy/default-backend-deploy -n emrys-prod

rollout-auth:
	kubectl set image deploy/auth-deploy auth-container=gcr.io/emrys-12/auth:latest -n emrys-prod
	kubectl rollout status deploy/auth-deploy -n emrys-prod

rollout-user:
	kubectl set image deploy/user-deploy user-container=gcr.io/emrys-12/user:latest -n emrys-prod
	kubectl rollout status deploy/user-deploy -n emrys-prod

rollout-miner:
	kubectl set image deploy/miner-deploy miner-container=gcr.io/emrys-12/miner:latest -n emrys-prod
	kubectl rollout status deploy/miner-deploy -n emrys-prod

rollout-job:
	kubectl set image sts/job-sts job-container=gcr.io/emrys-12/job:latest -n emrys-prod
	kubectl rollout status sts/job-sts -n emrys-prod

rollout-notebook:
	kubectl set image deploy/notebook-deploy notebook-container=gcr.io/emrys-12/notebook:latest -n emrys-prod
	kubectl rollout status deploy/notebook-deploy -n emrys-prod

rollout-image:
	kubectl set image deploy/image-deploy image-container=gcr.io/emrys-12/image:latest -n emrys-prod
	kubectl rollout status deploy/image-deploy -n emrys-prod

rollout-registry:
	kubectl set image deploy/registry-deploy image-container=gcr.io/emrys-12/registry:latest -n emrys-prod
	kubectl rollout status deploy/registry-deploy -n emrys-prod

rollout-data:
	kubectl set image sts/data-sts data-container=gcr.io/emrys-12/data:latest -n emrys-prod
	kubectl rollout status sts/data-sts -n emrys-prod

rollout-devpi:
	kubectl set image sts/devpi-sts devpi-container=gcr.io/emrys-12/devpi:latest -n emrys-prod
	kubectl rollout status sts/devpi-sts -n emrys-prod


rollback: rollback-default-backend rollback-auth rollback-user rollback-miner rollback-job rollback-notebook rollback-image rollback-registry rollback-data

rollback-default-backend:
	kubectl rollout undo deploy/default-backend-deploy -n emrys-prod
	kubectl rollout status deploy/default-backend-deploy -n emrys-prod

rollback-auth:
	kubectl rollout undo deploy/auth-deploy -n emrys-prod
	kubectl rollout status deploy/auth-deploy -n emrys-prod

rollback-user:
	kubectl rollout undo deploy/user-deploy -n emrys-prod
	kubectl rollout status deploy/user-deploy -n emrys-prod

rollback-miner:
	kubectl rollout undo deploy/miner-deploy -n emrys-prod
	kubectl rollout status deploy/miner-deploy -n emrys-prod

rollback-job:
	kubectl rollout undo sts/job-sts -n emrys-prod
	kubectl rollout status sts/job-sts -n emrys-prod

rollback-notebook:
	kubectl rollout undo deploy/notebook-deploy -n emrys-prod
	kubectl rollout status deploy/notebook-deploy -n emrys-prod

rollback-image:
	kubectl rollout undo deploy/image-deploy -n emrys-prod
	kubectl rollout status deploy/image-deploy -n emrys-prod

rollback-registry:
	kubectl rollout undo deploy/registry-deploy -n emrys-prod
	kubectl rollout status deploy/registry-deploy -n emrys-prod

rollback-data:
	kubectl rollout undo deploy/data-deploy -n emrys-prod
	kubectl rollout status deploy/data-deploy -n emrys-prod

rollback-devpi:
	kubectl rollout undo sts/devpi-sts -n emrys-prod
	kubectl rollout status sts/devpi-sts -n emrys-prod


delete: delete-default-backend delete-auth delete-user delete-miner delete-job delete-notebook delete-image delete-registry delete-data

delete-default-backend:
	kubectl delete pod -lapp=default-backend -n emrys-prod

delete-auth:
	kubectl delete pod -lapp=auth -n emrys-prod

delete-user:
	kubectl delete pod -lapp=user -n emrys-prod

delete-miner:
	kubectl delete pod -lapp=miner -n emrys-prod

delete-job:
	kubectl delete pod -lapp=job -n emrys-prod

delete-notebook:
	kubectl delete pod -lapp=notebook -n emrys-prod

delete-image:
	kubectl delete pod -lapp=image -n emrys-prod

delete-registry:
	kubectl delete pod -lapp=registry -n emrys-prod

delete-data:
	kubectl delete pod -lapp=data -n emrys-prod

delete-devpi:
	kubectl delete pod -lapp=devpi -n emrys-prod


patch-default-backend:
	kubectl patch deploy default-backend-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-auth:
	kubectl patch deploy auth-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-user:
	kubectl patch deploy user-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-miner:
	kubectl patch deploy miner-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-job:
	kubectl patch deploy job-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-notebook:
	kubectl patch deploy notebook-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-image:
	kubectl patch deploy image-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-registry:
	kubectl patch deploy registry-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-data:
	kubectl patch deploy data-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"

patch-devpi:
	kubectl patch deploy devpi-deploy -n emrys-prod -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"
