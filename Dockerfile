# Building Backend
FROM golang:alpine as messaging-server

RUN apk add nodejs npm

WORKDIR /source
COPY . .
WORKDIR /source/pkg/views
RUN npm install
RUN npm run build
WORKDIR /source
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs -o /dist ./pkg/cmd/main.go

# Runtime
FROM golang:alpine

COPY --from=messaging-server /dist /messaging/server

EXPOSE 8447

CMD ["/messaging/server"]
