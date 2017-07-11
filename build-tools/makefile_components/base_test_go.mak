# test section of Makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.

TESTOS = $(BUILD_OS)

test: build
	@mkdir -p bin/linux
	@mkdir -p $(GOTMP)/{src/$(PKG),pkg,bin,std/linux}
	@echo "Testing $(SRC_AND_UNDER) with TESTARGS=$(TESTARGS)"
	@docker run -t --rm  -u $(shell id -u):$(shell id -g)                 \
	    -v $(PWD)/$(GOTMP):/go                                                 \
	    -v $(PWD):/go/src/$(PKG)                                          \
	    -v $(PWD)/bin/linux:/go/bin                                     \
	    -v $(PWD)/$(GOTMP)/std/linux:/usr/local/go/pkg/linux_amd64_static  \
	    -e CGO_ENABLED=0	\
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
        go test -v -installsuffix static -ldflags '$(LDFLAGS)' $(SRC_AND_UNDER) $(TESTARGS)

# test_precompile allows a full compilation of _test.go files, without execution of the tests.
# Setup and teardown in TestMain is still executed though, so this can cost some time.
test_precompile: TESTARGS=-run '^$$'
test_precompile: test
