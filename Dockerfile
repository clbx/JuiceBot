FROM --platform=amd64 golang AS builder 
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY * ./
RUN go build

FROM --platform=amd64 golang
ENV TOKEN ${TOKEN}

LABEL "org.opencontainers.image.source" "https://github.com/clbx/juicebot" \
    "org.opencontainers.image.version" "0.1.0" \
    "org.opencontainers.image.base.name" "golang" 

COPY --from=builder /app/juicebot juicebot
CMD ./juicebot
