FROM golang:1.17-alpine AS build-env
ADD . /src
RUN apk add --no-cache git
ARG GIT_TOKEN
RUN git config --global url."https://${GIT_TOKEN}@dev.azure.com/bloopi/bloopi/_git/shared_models".insteadOf "https://dev.azure.com/bloopi/bloopi/_git/shared_models"
RUN cd /src && CGO_ENABLED=0 go build -a -o cli/agent/agent cli/agent/main.go

FROM alpine:latest

COPY --from=build-env /src/cli/agent /app/agent

CMD [ "/app/agent" ]