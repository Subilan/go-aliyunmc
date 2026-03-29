# hurl/users/Makefile
HURL=hurl --variables-file ../env/dev.env
USER ?= testuser

.PHONY: test-login test-delete test-create test-profile test-logout test-all report

test-login:
	$(HURL) login.hurl --variable username=$(USER)

test-delete:
	$(HURL) delete.hurl --variable username=$(USER)

test-create:
	$(HURL) first-create.hurl --variable username=$(USER)

test-profile:
	$(HURL) profile.hurl --variable username=$(USER)

test-logout:
	$(HURL) logout.hurl --variable username=$(USER)

# report:
# 	$(HURL) --report-html report login.hurl delete.hurl first-create.hurl profile.hurl logout.hurl --variable username=$(USER)
