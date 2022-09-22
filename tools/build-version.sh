#!/bin/bash

NOW=$(date -u "+%Y%m%dT%H%M" | tr -d "\n")
GIT_HEAD=$( git rev-parse head | tr -d "\n")

if git diff HEAD --exit-code --quiet ; then
  # git working dir is clean.
  IMAGE_VERSION="$GIT_HEAD"
else
  # git working dir is dirty, append "dirty" and the timestamp
  IMAGE_VERSION="$GIT_HEAD-dirty-${NOW}"
fi

echo -n "$IMAGE_VERSION"