FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD templates/*.yaml /templates/
ADD templates/rhsso/* /templates/rhsso/
ADD resources/* /resources/
ADD build/_output/bin/rhpam-dev-operator /usr/local/bin/rhpam-dev-operator
