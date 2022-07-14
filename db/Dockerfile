FROM golang:1.18.3-alpine3.16

ADD ./scripts ./scripts
ADD ./migrations ./migrations
ADD ./seeds ./seeds

RUN chmod +x ./scripts/db_migrate.sh

RUN apk add --update postgresql-client
RUN apk add --no-cache --virtual git

ARG DB_HOST
ARG DB_USER
ARG DB_PASSWORD

ENV DB_HOST=$DB_HOST
ENV DB_USER=$DB_USER
ENV DB_PASSWORD=$DB_PASSWORD

ARG TAG=${TAG:-"v4.14.1"}

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$TAG

CMD ["./scripts/db_migrate.sh","up"]
