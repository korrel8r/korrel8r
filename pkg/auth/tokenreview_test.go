// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package auth

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// reactor returns a fake reactor that authenticates tokens and tracks call count.
func reactor(uid, username string) (ktesting.ReactionFunc, *atomic.Int32) {
	var count atomic.Int32
	return func(action ktesting.Action) (bool, runtime.Object, error) {
		count.Add(1)
		tr := action.(ktesting.CreateAction).GetObject().(*authenticationv1.TokenReview)
		tr.Status = authenticationv1.TokenReviewStatus{
			Authenticated: true,
			User:          authenticationv1.UserInfo{UID: uid, Username: username},
		}
		return true, tr, nil
	}, &count
}

func TestTokenReview_Authenticated(t *testing.T) {
	cs := fake.NewSimpleClientset()
	react, _ := reactor("uid-123", "testuser")
	cs.PrependReactor("create", "tokenreviews", react)

	tr := &TokenReview{clientset: cs}
	key, err := tr.User("my-token")
	require.NoError(t, err)
	assert.Equal(t, "testuser", key)
}

func TestTokenReview_SameUsername_DifferentTokens(t *testing.T) {
	cs := fake.NewSimpleClientset()
	react, _ := reactor("uid-123", "testuser")
	cs.PrependReactor("create", "tokenreviews", react)

	tr := &TokenReview{clientset: cs}
	key1, err := tr.User("token-A")
	require.NoError(t, err)
	key2, err := tr.User("token-B")
	require.NoError(t, err)
	assert.Equal(t, key1, key2, "same username should produce the same session key")
}

func TestTokenReview_NotAuthenticated(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "tokenreviews", func(action ktesting.Action) (bool, runtime.Object, error) {
		tr := action.(ktesting.CreateAction).GetObject().(*authenticationv1.TokenReview)
		tr.Status = authenticationv1.TokenReviewStatus{Authenticated: false}
		return true, tr, nil
	})

	tr := &TokenReview{clientset: cs}
	key, err := tr.User("bad-token")
	assert.Error(t, err)
	assert.Empty(t, key)
}

func TestTokenReview_Error(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "tokenreviews", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("network error")
	})

	tr := &TokenReview{clientset: cs}
	key, err := tr.User("some-token")
	assert.Error(t, err)
	assert.Empty(t, key)
}

func TestTokenReview_TokenPassedThrough(t *testing.T) {
	cs := fake.NewSimpleClientset()
	var receivedToken string
	cs.PrependReactor("create", "tokenreviews", func(action ktesting.Action) (bool, runtime.Object, error) {
		tr := action.(ktesting.CreateAction).GetObject().(*authenticationv1.TokenReview)
		receivedToken = tr.Spec.Token
		tr.Status = authenticationv1.TokenReviewStatus{
			Authenticated: true,
			User:          authenticationv1.UserInfo{UID: "uid-456", Username: "user"},
		}
		return true, tr, nil
	})

	tr := &TokenReview{clientset: cs}
	_, err := tr.User("my-secret-token")
	require.NoError(t, err)
	assert.Equal(t, "my-secret-token", receivedToken, "token should be passed through as-is")
}

func TestTokenReview_CacheHit(t *testing.T) {
	cs := fake.NewSimpleClientset()
	react, count := reactor("uid-123", "testuser")
	cs.PrependReactor("create", "tokenreviews", react)

	tr := &TokenReview{clientset: cs}
	u1, err := tr.User("my-token")
	require.NoError(t, err)
	assert.Equal(t, "testuser", u1)
	assert.Equal(t, int32(1), count.Load())

	u2, err := tr.User("my-token")
	require.NoError(t, err)
	assert.Equal(t, "testuser", u2)
	assert.Equal(t, int32(1), count.Load(), "second call should use cache, not call API")
}

func TestTokenReview_CacheDifferentTokens(t *testing.T) {
	cs := fake.NewSimpleClientset()
	react, count := reactor("uid-123", "testuser")
	cs.PrependReactor("create", "tokenreviews", react)

	tr := &TokenReview{clientset: cs}
	_, err := tr.User("token-A")
	require.NoError(t, err)
	_, err = tr.User("token-B")
	require.NoError(t, err)
	assert.Equal(t, int32(2), count.Load(), "different tokens should each call API")

	_, err = tr.User("token-A")
	require.NoError(t, err)
	_, err = tr.User("token-B")
	require.NoError(t, err)
	assert.Equal(t, int32(2), count.Load(), "repeated calls should use cache")
}

func TestTokenReview_EmptyUsername(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "tokenreviews", func(action ktesting.Action) (bool, runtime.Object, error) {
		tr := action.(ktesting.CreateAction).GetObject().(*authenticationv1.TokenReview)
		tr.Status = authenticationv1.TokenReviewStatus{
			Authenticated: true,
			User:          authenticationv1.UserInfo{UID: "uid-fallback", Username: ""},
		}
		return true, tr, nil
	})

	tr := &TokenReview{clientset: cs}
	key, err := tr.User("token")
	require.NoError(t, err)
	assert.Equal(t, "uid-fallback", key, "should fall back to UID when username is empty")
}
