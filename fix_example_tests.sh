#!/bin/bash

components=("etcd" "mysql" "postgres" "mongodb" "redis")

for comp in "${components[@]}"; do
    echo "Processing component: $comp"
    dir="pkg/component/$comp"
    
    # Find example_test.go
    file="$dir/example_test.go"
    if [ -f "$file" ]; then
        echo "Fixing $file"
        
        # Add import if missing
        if ! grep -q "github.com/kart-io/sentinel-x/pkg/component/$comp" "$file"; then
            # Insert import after package declaration or imports
            # Simple approach: replace "import (" with "import (\n\t\"github.com/kart-io/sentinel-x/pkg/component/$comp\""
            sed -i "s/import (/import (\n\t\"github.com\/kart-io\/sentinel-x\/pkg\/component\/$comp\"/" "$file"
        fi
        
        # Replace NewOptions() with comp.NewOptions()
        sed -i "s/NewOptions()/$comp.NewOptions()/g" "$file"
        
        # Replace &Options with &comp.Options
        sed -i "s/&Options/&$comp.Options/g" "$file"
        
        # Replace *Options with *comp.Options
        sed -i "s/\*Options/*$comp.Options/g" "$file"
        
        # Replace opts.OptionField with opts.OptionField (no change needed)
        
        # Replace undefined: comp with nothing (if it was compOpts)
        # But wait, if I replaced compOpts with nothing, then compOpts.NewOptions became NewOptions.
        # Now I replace NewOptions with comp.NewOptions.
        # This seems correct.
        
        # Also fix aliases if any remain
        alias="${comp}Opts"
        if [ "$comp" == "postgres" ]; then alias="pgOpts"; fi
        if [ "$comp" == "mongodb" ]; then alias="mongoOpts"; fi
        
        sed -i "s/$alias\.//g" "$file"
    fi
done
