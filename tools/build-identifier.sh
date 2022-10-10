#!/bin/bash
##
# build-identifier uses the HEAD Git SHA to provide a unique id number for a build.
# If the working directory is dirty, it will append the current timestamp
# to the HEAD Git SHA so that the build identifier is unique.
NOW=$(date -u "+%Y%m%dT%H%M" | tr -d "\n")
GIT_HEAD=$( git rev-parse HEAD | tr -d "\n")

if git diff HEAD --exit-code --quiet ; then
  # git working dir is clean.
  IMAGE_VERSION="$GIT_HEAD"
else
  # git working dir is dirty, append "dirty" and the timestamp
  IMAGE_VERSION="$GIT_HEAD-dirty-${NOW}"
fi

echo -n "$IMAGE_VERSION"
