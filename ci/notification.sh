#!/bin/sh

DATA="{'repo': $GITHUB_REPOSITORY, \
       'commit': $GITHUB_SHA, \
       'workflow': $GITHUB_WORKFLOW \
       'status': $ACTIONS_STATUS}"
CURL_NETWORK="--connect-timeout 10 --max-time 10 --retry 3 --retry-delay 0 --retry-max-time 60"
exec curl $CURL_NETWORK -sSL -D - -H "Auth: $WEBHOOK_AUTH" -d "$DATA" "$WEBHOOK_URL" -o /dev/null
