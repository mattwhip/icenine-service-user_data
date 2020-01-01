# This is a multi-stage Dockerfile and requires >= Docker 17.05
# https://docs.docker.com/engine/userguide/eng-image/multistage-build/
ARG GC_PROJECT

FROM gcr.io/${GC_PROJECT}/gobuild:latest as builder

# Copy project files into container
ADD . .

# Uncomment for resolution of previously prepped local dependencies with 'go mod vendor'
# COPY vendor/bitbucket.org/gopileon/icenine-database ../icenine-database
# COPY vendor/bitbucket.org/gopileon/icenine-services ../icenine-services

RUN git config --global credential.helper 'store --file ~/gobuffalo/.gitcredentials'

# Build the app, and statically link all dependencies
RUN buffalo build --ldflags '-extldflags "-static"' -o /bin/app


FROM alpine:3.8

WORKDIR /bin/

COPY --from=builder /bin/app .

# Bind the app to 0.0.0.0 so it can be seen from outside the container
ENV ADDR=0.0.0.0

# Comment out to run the migrations before running the binary:
# CMD /bin/app migrate; /bin/app
CMD exec /bin/app
