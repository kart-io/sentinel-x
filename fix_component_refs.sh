#!/bin/bash

components=("etcd" "mysql" "postgres" "mongodb" "redis")

for comp in "${components[@]}"; do
    echo "Processing component: $comp"
    dir="pkg/component/$comp"
    
    # Find all go files in the component directory
    find "$dir" -name "*.go" | while read file; do
        # Skip _test.go files for now (handled separately or manually)
        if [[ "$file" == *"_test.go" ]]; then continue; fi
        
        echo "Fixing $file"
        
        # Replace comp.Type with Type
        # e.g. etcd.Client -> Client
        # e.g. mysql.Options -> Options
        
        sed -i "s/$comp\.//g" "$file"
        
        # Also handle specific aliases if any
        if [ "$comp" == "postgres" ]; then
            sed -i "s/postgres\.//g" "$file"
        elif [ "$comp" == "mongodb" ]; then
            sed -i "s/mongodb\.//g" "$file"
        fi
    done
done
