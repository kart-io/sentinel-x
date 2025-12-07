.PHONY: update
update:
	bash hack/update-vendor.sh

.PHONY: update-goagent
update-goagent:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: publish-goagent
publish-goagent:
	bash hack/sync-to-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: update-logger
update-logger:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main

.PHONY: publish-logger
publish-logger:
	bash hack/sync-to-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main
