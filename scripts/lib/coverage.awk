#!/usr/bin/awk -f

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# usage: go tool cover -func=coverage.out | awk -v target=80 -f coverage.awk

{
    if ($1 == "total:") {
        print "Total coverage: " $3
        val = $3
        sub(/%/, "", val)
        if (val < target) {
            print "Coverage (" val "%) is below the target (" target "%)"
            exit 1
        }
    }
}
