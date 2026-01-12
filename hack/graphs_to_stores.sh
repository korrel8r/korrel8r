#!/bin/bash
#!/bin/bash
# Extract queries from JSON result graph(s), generate mock store YAML files.
# Use to create mock store files for off-line testing, based on real cluster data.
# Will sort and uniq the input.

# Extract queries from JSON result graph(s)
jq -r '.nodes[].queries[].query' | sort -u | while read -r QUERY; do
	# Key must be quoted.
	echo "$QUERY" | jq -Rr '@json + ":"'
	korrel8r objects "$QUERY"
done
