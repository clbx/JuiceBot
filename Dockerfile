FROM --platform=linux/amd64 golang:1.24.7 AS builder 
WORKDIR /app
# COPY go.mod ./
# COPY go.sum ./
COPY . .
RUN go mod download
RUN go build .

FROM --platform=linux/amd64 debian:bookworm-slim
#ENV TOKEN ${TOKEN}

LABEL "org.opencontainers.image.source"="https://github.com/clbx/juicebot" \
    "org.opencontainers.image.version"="0.1.0" \
    "org.opencontainers.image.base.name"="golang" 

RUN apt update
RUN apt install ca-certificates -y

COPY --from=builder /app/juicebot juicebot
CMD ./juicebot
