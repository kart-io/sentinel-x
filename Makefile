.PHONY: update-staging
update-staging:
	bash hack/update-staging.sh

.PHONY: update
update: update-staging
	bash hack/update-vendor.sh
