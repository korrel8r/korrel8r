// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const cacheExpiry = 5 * time.Minute

type cacheEntry struct {
	username string
	expires  time.Time
}

// TokenReview resolves bearer tokens to Kubernetes usernames via the TokenReview API.
//
// It uses korrel8r's own service account (not the request token) since
// TokenReview is a privileged operation.
// Results are cached to avoid repeated API calls for the same token.
type TokenReview struct {
	clientset kubernetes.Interface
	cache     sync.Map // map[string]cacheEntry — keyed by bearer token
}

// NewTokenReviewFromClientset creates a TokenReview from an existing Kubernetes clientset.
func NewTokenReviewFromClientset(cs kubernetes.Interface) *TokenReview {
	return &TokenReview{clientset: cs}
}

// NewTokenReview creates a TokenReview using in-cluster or kubeconfig credentials.
func NewTokenReview() (*TokenReview, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	// Check that the service account has permission to create TokenReviews.
	// A "forbidden" error means the SA can't create tokenreviews and never will.
	// A "not authenticated" response is fine — it means we have RBAC access.
	probe := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{Token: "probe"},
	}
	_, err = clientset.AuthenticationV1().TokenReviews().Create(
		context.Background(), probe, metav1.CreateOptions{})
	if apierrors.IsForbidden(err) {
		return nil, err
	}
	return &TokenReview{clientset: clientset}, nil
}

// User resolves a bearer token to a Kubernetes username.
// Returns the username, falling back to UID if username is empty.
// Results are cached for 5 minutes per token.
func (tr *TokenReview) User(token string) (string, error) {
	if v, ok := tr.cache.Load(token); ok {
		if e := v.(cacheEntry); time.Now().Before(e.expires) {
			return e.username, nil
		}
		tr.cache.Delete(token)
	}
	review := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{Token: token},
	}
	result, err := tr.clientset.AuthenticationV1().TokenReviews().Create(
		context.Background(), review, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("TokenReview failed: %w", err)
	}
	if !result.Status.Authenticated {
		return "", fmt.Errorf("TokenReview: token not authenticated")
	}
	username := result.Status.User.Username
	if username == "" {
		username = result.Status.User.UID
	}
	if username == "" {
		return "", fmt.Errorf("TokenReview: no username or UID in response")
	}
	tr.cache.Store(token, cacheEntry{username: username, expires: time.Now().Add(cacheExpiry)})
	return username, nil
}
