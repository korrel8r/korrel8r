package console

import (
	"testing"

	alert "github.com/korrel8/korrel8/pkg/amalert"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
)

func TestURLs(t *testing.T) {
	for _, x := range []struct {
		url   string
		class korrel8.Class
		query string
	}{
		{
			url:   "https://console-openshift-console.apps.snoflake.my.test/monitoring/alerts/354014176?alertname=KubeDeploymentReplicasMismatch&container=kube-rbac-proxy-main&deployment=httpd&endpoint=https-main&job=kube-state-metrics&namespace=openshift-demo&service=kube-state-metrics&severity=warning",
			class: alert.Domain.Class("alert"),
			query: "?filter=alertname%3DKubeDeploymentReplicasMismatch",
		},
	} {
		c, ref, err := ParseURL(x.url)
		assert.NoError(t, err)
		assert.Equal(t, x.class, c)
		assert.Equal(t, x.query, ref.String())
	}
}
