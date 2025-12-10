#!/bin/bash
# Interactive script to generate Korrel8r correlation rules

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Korrel8r Rule Generator ===${NC}\n"

# Common domains
DOMAINS=("k8s" "log" "metric" "alert" "trace" "netflow" "incident")

# Helper function to display domain info
show_domain_info() {
    echo -e "${YELLOW}Common domains:${NC}"
    echo "  k8s       - Kubernetes resources (Pod, Deployment.apps, Node, etc.)"
    echo "  log       - Application logs"
    echo "  metric    - Prometheus metrics"
    echo "  alert     - Alertmanager alerts"
    echo "  trace     - Distributed traces"
    echo "  netflow   - Network flow data"
    echo "  incident  - Incident management"
    echo ""
}

# Helper function to display template function info
show_template_functions() {
    echo -e "${YELLOW}Available template functions:${NC}"
    echo "  {{.metadata.namespace}}, {{.metadata.name}} - K8s object fields"
    echo "  {{mustToJson .}}                            - Convert to JSON"
    echo "  {{k8sClass .apiVersion .kind}}              - Generate K8s class name"
    echo "  {{lower .kind}}                             - Lowercase string"
    echo "  {{logTypeForNamespace .metadata.namespace}} - Get log type"
    echo ""
}

# Step 1: Rule name
echo -e "${GREEN}Step 1: Rule Name${NC}"
read -p "Enter rule name (e.g., PodToLogs, AlertToDeployment): " RULE_NAME

# Step 2: Source domain
echo -e "\n${GREEN}Step 2: Source (Start) Configuration${NC}"
show_domain_info
read -p "Enter source domain: " SOURCE_DOMAIN

read -p "Enter source classes (comma-separated, or leave empty for all): " SOURCE_CLASSES
if [ -n "$SOURCE_CLASSES" ]; then
    SOURCE_CLASSES_YAML="
    classes: [${SOURCE_CLASSES}]"
fi

# Step 3: Target domain
echo -e "\n${GREEN}Step 3: Target (Goal) Configuration${NC}"
show_domain_info
read -p "Enter target domain: " TARGET_DOMAIN

read -p "Enter target classes (comma-separated, or leave empty for all): " TARGET_CLASSES
if [ -n "$TARGET_CLASSES" ]; then
    TARGET_CLASSES_YAML="
    classes: [${TARGET_CLASSES}]"
fi

# Step 4: Query template
echo -e "\n${GREEN}Step 4: Query Template${NC}"
show_template_functions
echo "Examples:"
echo "  k8s:Pod:{namespace: \"{{.metadata.namespace}}\", name: \"{{.metadata.name}}\"}"
echo "  log:{{logTypeForNamespace .metadata.namespace}}:{\"namespace\":\"{{.metadata.namespace}}\"}"
echo "  metric:metric:{namespace=\"{{.metadata.namespace}}\",{{lower .kind}}=\"{{.metadata.name}}\"}"
echo ""
read -p "Enter query template: " QUERY_TEMPLATE

# Step 5: Target file
echo -e "\n${GREEN}Step 5: Target File${NC}"
echo "Existing rule files:"
ls -1 etc/korrel8r/rules/*.yaml 2>/dev/null | grep -v "_test" | sed 's|etc/korrel8r/rules/||' || echo "  (none found)"
echo ""
read -p "Enter target filename (e.g., k8s.yaml, or new-file.yaml): " TARGET_FILE

# Ensure .yaml extension
if [[ ! "$TARGET_FILE" =~ \.yaml$ ]]; then
    TARGET_FILE="${TARGET_FILE}.yaml"
fi

FULL_PATH="etc/korrel8r/rules/${TARGET_FILE}"

# Generate the rule YAML
RULE_YAML="  - name: ${RULE_NAME}
    start:
      domain: ${SOURCE_DOMAIN}${SOURCE_CLASSES_YAML}
    goal:
      domain: ${TARGET_DOMAIN}${TARGET_CLASSES_YAML}
    result:
      query: |-
        ${QUERY_TEMPLATE}
"

# Display preview
echo -e "\n${BLUE}=== Generated Rule ===${NC}"
echo "$RULE_YAML"
echo ""

# Confirm
read -p "Add this rule to ${FULL_PATH}? (y/n): " CONFIRM

if [[ "$CONFIRM" =~ ^[Yy]$ ]]; then
    # Check if file exists
    if [ -f "$FULL_PATH" ]; then
        # Append to existing file
        if ! grep -q "^rules:" "$FULL_PATH"; then
            echo "rules:" >> "$FULL_PATH"
        fi
        echo "$RULE_YAML" >> "$FULL_PATH"
        echo -e "${GREEN}✓ Rule appended to ${FULL_PATH}${NC}"
    else
        # Create new file
        cat > "$FULL_PATH" <<EOF
rules:
$RULE_YAML
EOF
        echo -e "${GREEN}✓ Created new file ${FULL_PATH}${NC}"
    fi

    echo -e "\n${YELLOW}Next steps:${NC}"
    echo "  1. Review the generated rule in ${FULL_PATH}"
    echo "  2. Run tests: go test ./etc/korrel8r/rules/..."
    echo "  3. Test the rule with korrel8r CLI or API"
else
    echo -e "${YELLOW}Rule generation cancelled${NC}"
    echo "You can copy the rule above and add it manually if needed."
fi
