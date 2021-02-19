VERSION_GITHASH = $(shell git rev-parse --short HEAD)
GO_LDFLAGS = CGO_ENABLED=0 go build -ldflags "-X main.build=${VERSION_GITHASH}" -a -tags netgo
DEPLOY_USER = $(shell echo "$${DEPLOY_USER}")
DEPLOY_IP = $(shell echo "$${DEPLOY_IP}")

.PHONY: all

all: build deploy

build:
	$(GO_LDFLAGS) -o go-ampedge .

clean:
	rm -rf go-ampedge

deploy:
	ssh -i ~/.ssh/deploy -o StrictHostKeyChecking=no  $(DEPLOY_USER)@$(DEPLOY_IP) 'sudo systemctl stop go-ampedge'
	for d in go-ampedge config.json; do \
		scp -i ~/.ssh/deploy -o StrictHostKeyChecking=no -r $$d $(DEPLOY_USER)@$(DEPLOY_IP):/home/$(DEPLOY_USER)/app/; \
	done
	ssh -i ~/.ssh/deploy -o StrictHostKeyChecking=no  $(DEPLOY_USER)@$(DEPLOY_IP) 'sudo systemctl start go-ampedge'
