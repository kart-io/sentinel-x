#!/bin/bash

components=("redis" "mysql" "postgres" "mongodb" "etcd")

for comp in "${components[@]}"; do
    echo "Processing component: $comp"
    dir="pkg/component/$comp"
    
    # Find files in the component directory
    find "$dir" -name "*.go" | while read file; do
        # Check if file imports itself
        if grep -q "github.com/kart-io/sentinel-x/pkg/component/$comp" "$file"; then
            echo "Fixing $file"
            
            # Remove the import line
            sed -i "/github.com\/kart-io\/sentinel-x\/pkg\/component\/$comp/d" "$file"
            
            # Replace usages
            # e.g. redisOpts.Options -> Options
            # e.g. redisOpts.NewOptions -> NewOptions
            # We need to handle different alias names if possible, but usually it's ${comp}Opts
            
            # Common aliases: redisOpts, mysqlOpts, pgOpts (special case), mongoOpts (special case), etcdOpts
            
            alias="${comp}Opts"
            if [ "$comp" == "postgres" ]; then
                alias="pgOpts"
            elif [ "$comp" == "mongodb" ]; then
                alias="mongoOpts" # or mongodbOpts
            fi
            
            # Replace alias.Type with Type
            sed -i "s/$alias\.//g" "$file"
            
            # Also handle mongodbOpts if it exists
            if [ "$comp" == "mongodb" ]; then
                sed -i "s/mongodbOpts\.//g" "$file"
            fi
        fi
    done
done
