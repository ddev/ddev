FROM golang:1.7-alpine
ENV DRUD_BUILD_DIR $GOPATH/src/github.com/drud/bootstrap/cli
ENV GLIDE_VERSION 0.11.1
ENV GLIDE_SHA256 de0c7870738c6bc11128761d53a99ad68687b0a213fe52cea15ad05d93f10e42
ENV CGO_ENABLED=0 
ENV GOOS=linux 

ADD https://github.com/Masterminds/glide/releases/download/v0.12.0/glide-v0.12.0-linux-amd64.tar.gz /glide.tar.gz
RUN apk add  --update bzr git ca-certificates wget gcc abuild binutils binutils-doc gcc-doc cmake cmake-doc bash musl-dev openssl \
    && mkdir -p $DRUD_BUILD_DIR \
    && wget https://github.com/Masterminds/glide/releases/download/v${GLIDE_VERSION}/glide-v${GLIDE_VERSION}-linux-amd64.tar.gz \
    && echo "${GLIDE_SHA256} */glide-v${GLIDE_VERSION}-linux-amd64.tar.gz" \
    && tar -zxf glide-v${GLIDE_VERSION}-linux-amd64.tar.gz \
    && mv linux-amd64/glide /usr/bin \
    && rm -rf linux-amd64 glide-v${GLIDE_VERSION}-linux-amd64.tar.gz


WORKDIR $DRUD_BUILD_DIR

ADD . $DRUD_BUILD_DIR

RUN glide install \
    && go get .

RUN go build -a -installsuffix cgo -ldflags '-w' -o $GOPATH/bin/drud  ./main.go

# We repeat this as a cmd so you can volume mount in a bin directory to generate a binary.
CMD ["go", "build", "-a", "-installsuffix", "cgo", "-ldflags", "'-w'", "-o", "go/bin/drud", "./main.go"] 