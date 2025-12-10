#!/bin/bash

# Define replacements
declare -A replacements=(
    ["github.com/kart-io/sentinel-x/pkg/app"]="github.com/kart-io/sentinel-x/pkg/infra/app"
    ["github.com/kart-io/sentinel-x/pkg/server"]="github.com/kart-io/sentinel-x/pkg/infra/server"
    ["github.com/kart-io/sentinel-x/pkg/datasource"]="github.com/kart-io/sentinel-x/pkg/infra/datasource"
    ["github.com/kart-io/sentinel-x/pkg/bridge"]="github.com/kart-io/sentinel-x/pkg/infra/adapter"
    ["github.com/kart-io/sentinel-x/pkg/middleware"]="github.com/kart-io/sentinel-x/pkg/infra/middleware"
    ["github.com/kart-io/sentinel-x/pkg/options/logger"]="github.com/kart-io/sentinel-x/pkg/infra/logger"
    ["github.com/kart-io/sentinel-x/pkg/options/server"]="github.com/kart-io/sentinel-x/pkg/infra/server"
    ["github.com/kart-io/sentinel-x/pkg/options/grpc"]="github.com/kart-io/sentinel-x/pkg/infra/server/grpc"
    ["github.com/kart-io/sentinel-x/pkg/options/http"]="github.com/kart-io/sentinel-x/pkg/infra/server/http"
    ["github.com/kart-io/sentinel-x/pkg/redis"]="github.com/kart-io/sentinel-x/pkg/component/redis"
    ["github.com/kart-io/sentinel-x/pkg/options/redis"]="github.com/kart-io/sentinel-x/pkg/component/redis"
    ["github.com/kart-io/sentinel-x/pkg/mysql"]="github.com/kart-io/sentinel-x/pkg/component/mysql"
    ["github.com/kart-io/sentinel-x/pkg/options/mysql"]="github.com/kart-io/sentinel-x/pkg/component/mysql"
    ["github.com/kart-io/sentinel-x/pkg/mongodb"]="github.com/kart-io/sentinel-x/pkg/component/mongodb"
    ["github.com/kart-io/sentinel-x/pkg/options/mongodb"]="github.com/kart-io/sentinel-x/pkg/component/mongodb"
    ["github.com/kart-io/sentinel-x/pkg/postgres"]="github.com/kart-io/sentinel-x/pkg/component/postgres"
    ["github.com/kart-io/sentinel-x/pkg/options/postgres"]="github.com/kart-io/sentinel-x/pkg/component/postgres"
    ["github.com/kart-io/sentinel-x/pkg/etcd"]="github.com/kart-io/sentinel-x/pkg/component/etcd"
    ["github.com/kart-io/sentinel-x/pkg/options/etcd"]="github.com/kart-io/sentinel-x/pkg/component/etcd"
    ["github.com/kart-io/sentinel-x/pkg/storage"]="github.com/kart-io/sentinel-x/pkg/component/storage"
    ["github.com/kart-io/sentinel-x/pkg/auth"]="github.com/kart-io/sentinel-x/pkg/security/auth"
    ["github.com/kart-io/sentinel-x/pkg/options/jwt"]="github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
    ["github.com/kart-io/sentinel-x/pkg/authz"]="github.com/kart-io/sentinel-x/pkg/security/authz"
    ["github.com/kart-io/sentinel-x/pkg/errors"]="github.com/kart-io/sentinel-x/pkg/utils/errors"
    ["github.com/kart-io/sentinel-x/pkg/validator"]="github.com/kart-io/sentinel-x/pkg/utils/validator"
    ["github.com/kart-io/sentinel-x/pkg/id"]="github.com/kart-io/sentinel-x/pkg/utils/id"
    ["github.com/kart-io/sentinel-x/pkg/response"]="github.com/kart-io/sentinel-x/pkg/utils/response"
)

# Find all go files
find . -name "*.go" | while read file; do
    echo "Processing $file"
    for old in "${!replacements[@]}"; do
        new="${replacements[$old]}"
        sed -i "s|$old|$new|g" "$file"
    done
done
