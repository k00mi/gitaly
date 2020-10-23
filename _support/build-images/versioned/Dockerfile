ARG CUSTOM_IMAGE_VERSION
FROM registry.gitlab.com/gitlab-org/gitlab-build-images:${CUSTOM_IMAGE_VERSION}

RUN mkdir -p /bundle-cache

COPY ./Gemfile /bundle-cache
COPY ./Gemfile.lock /bundle-cache

RUN DEBIAN_FRONTEND=noninteractive apt-get update -qq && \
    DEBIAN_FRONTEND=noninteractive apt-get install -qq -y rubygems bundler cmake build-essential libicu-dev && \
    cd /bundle-cache && gem install bundler -v 1.17.3 && bundle install --path vendor/bundle
