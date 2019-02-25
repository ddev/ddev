# test section of Makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.

TESTOS = $(BUILD_OS)

test: build
	@echo "Testing $(SRC_AND_UNDER) with TESTARGS=$(TESTARGS)"
	@mkdir -p $(GOTMP)/{.cache,pkg,src,bin}
	@$(DOCKERTESTCMD) \
        go test $(USEMODVENDOR) -v -installsuffix static -ldflags '$(LDFLAGS)' $(SRC_AND_UNDER) $(TESTARGS)
	$( shell if [ -d $(GOTMP) ]; then chmod -R u+w $(GOTMP); fi )

# test_precompile allows a full compilation of _test.go files, without execution of the tests.
# Setup and teardown in TestMain is still executed though, so this can cost some time.
test_precompile: TESTARGS=-run '^$$'
test_precompile: test
