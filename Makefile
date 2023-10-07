
build:
	./tf/build.sh

apply: build
	./tf/apply.sh

plan: build
	./tf/plan.sh

destroy: 
	./tf/destroy.sh

setup-dev:
	terraform -chdir=./tf workspace select default

setup-prod:
	terraform -chdir=./tf workspace select production