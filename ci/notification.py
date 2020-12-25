#!/usr/bin/env python3

import json
import os
import urllib.request

TIMEOUT = 10
RETRIES = 3


def main():
    data = {
        "repo": os.environ["GITHUB_REPOSITORY"],
        "commit": os.environ["GITHUB_SHA"],
        "workflow": os.environ["GITHUB_WORKFLOW"],
        "status": os.environ["ACTION_STATUS"]
    }
    req = urllib.request.Request(
        url=os.environ["WEBHOOK_URL"],
        data=json.dumps(data).encode("utf-8"),
        method="POST",
        headers={
            "Auth": os.environ["WEBHOOK_AUTH"],
            "Content-Type": "application/json"
        }
    )
    for i in range(RETRIES):
        try:
            f = urllib.request.urlopen(req, timeout=TIMEOUT)
            status = f.status
            if status != 200:
                raise ValueError(f"The request failed ({status})")
            print("success")
            return
        except Exception as e:
            print(f"{i} failed with exception: {e}")
            pass
    print("Failed to connect to webhook.")
    exit(-1)


if __name__ == "__main__":
    main()
