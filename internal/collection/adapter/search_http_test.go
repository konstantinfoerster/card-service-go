package adapter_test

import (
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter"
	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	oidcfakes "github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	"github.com/konstantinfoerster/card-service-go/internal/common/img"
	detectfakes "github.com/konstantinfoerster/card-service-go/internal/common/img/fakes"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	srv := searchServer(t)
	cases := []struct {
		name                string
		header              func(req *http.Request)
		page                string
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "default page as json",
			header:              func(req *http.Request) {},
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				expected := []adapter.Card{
					{Item: adapter.Item{ID: 434}, Name: "Demonic Attorney"},
					{Item: adapter.Item{ID: 706}, Name: "Demonic Hordes"},
					{Item: adapter.Item{ID: 514}, Name: "Demonic Tutor"},
				}
				body := commontest.FromJSON[adapter.PagedResult[adapter.Card]](t, resp)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name:                "second page as json",
			header:              func(req *http.Request) {},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				expected := []adapter.Card{
					{Item: adapter.Item{ID: 706}, Name: "Demonic Hordes"},
				}
				body := commontest.FromJSON[adapter.PagedResult[adapter.Card]](t, resp)
				assert.True(t, body.HasMore)
				assert.Equal(t, 2, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name: "default page as html",
			header: func(req *http.Request) {
				req.Header.Set(fiber.HeaderAccept, fiber.MIMETextHTMLCharsetUTF8)
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsFullHTML(t, body)
				commontest.AssertContainsLogin(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "default page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "second page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-"), "expected 1 card  in %s", body)
			},
		},
		{
			name: "htmx has lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			page:                "size=1",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				assert.Contains(t, body, "hidden-card")
			},
		},
		{
			name: "htmx has no lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				assert.NotContains(t, body, "hidden-card")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("http://localhost/cards?name=Demonic&"+tc.page),
			)
			tc.header(req)

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestSearchWithUser(t *testing.T) {
	srv := searchServer(t)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser.ID})
	cases := []struct {
		name                string
		header              func(req *http.Request)
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "default page as json",
			header:              func(req *http.Request) {},
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				expected := []adapter.Card{
					{Item: adapter.Item{ID: 434}, Name: "Demonic Attorney"},
					{Item: adapter.Item{ID: 706, Amount: 3}, Name: "Demonic Hordes"},
					{Item: adapter.Item{ID: 514, Amount: 5}, Name: "Demonic Tutor"},
				}
				body := commontest.FromJSON[adapter.PagedResult[adapter.Card]](t, resp)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name: "default page as html",
			header: func(req *http.Request) {
				req.Header.Set(fiber.HeaderAccept, fiber.MIMETextHTMLCharsetUTF8)
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsFullHTML(t, body)
				commontest.AssertContainsProfile(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
				assert.Equal(t, 3, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 2, strings.Count(body, "data-testid=\"remove-card-btn\""))
			},
		},
		{
			name: "default page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
				assert.Equal(t, 3, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 2, strings.Count(body, "data-testid=\"remove-card-btn\""))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("http://localhost/cards?name=Demonic"),
				commontest.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
			)
			tc.header(req)

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestSearchWithInvalidUser(t *testing.T) {
	srv := searchServer(t)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: invalidUser})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/cards?name=Demonic&size=5&page=1"),
		commontest.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}

func TestDetect(t *testing.T) {
	srv := searchServer(t)
	fImg, err := os.Open(path.Join(currentDir(), "testdata", "cardImageModified.jpg"))
	defer commonio.Close(fImg)
	require.NoError(t, err)

	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodPost),
		commontest.WithURL("http://localhost/detect"),
		commontest.WithMultipartFile(t, fImg, fImg.Name()),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	score := 4
	expected := []adapter.Card{
		{
			Item:  adapter.Item{ID: 1},
			Name:  "Ancestor's Chosen",
			Image: "cardImage.jpg",
			Score: &score,
		},
	}
	body := commontest.FromJSON[adapter.PagedResult[adapter.Card]](t, resp)
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.ElementsMatch(t, expected, body.Data)
}

func TestDetectWithUser(t *testing.T) {
	srv := searchServer(t)
	fImg, err := os.Open(path.Join(currentDir(), "testdata", "cardImageModified.jpg"))
	defer commonio.Close(fImg)
	require.NoError(t, err)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: detectorUser.ID})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodPost),
		commontest.WithURL("http://localhost/detect"),
		commontest.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
		commontest.WithMultipartFile(t, fImg, fImg.Name()),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	score := 4
	expected := []adapter.Card{
		{
			Item:  adapter.Item{ID: 1, Amount: 1},
			Name:  "Ancestor's Chosen",
			Image: "cardImage.jpg",
			Score: &score,
		},
	}
	body := commontest.FromJSON[adapter.PagedResult[adapter.Card]](t, resp)
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.ElementsMatch(t, expected, body.Data)
}
func searchServer(t *testing.T) *server.Server {
	srv := defaultServer()

	authSvc := oidcfakes.NewUserService(validUser, detectorUser)
	repo, err := fakes.NewRepository(img.NewPHasher())
	require.NoError(t, err)

	collectSvc := application.NewCollectionService(repo, repo)

	_, err = collectSvc.Collect(domain.Item{ID: 434, Amount: 0}, domain.Collector{ID: validUser.ID})
	require.NoError(t, err)
	_, err = collectSvc.Collect(domain.Item{ID: 514, Amount: 5}, domain.Collector{ID: validUser.ID})
	require.NoError(t, err)
	_, err = collectSvc.Collect(domain.Item{ID: 706, Amount: 3}, domain.Collector{ID: validUser.ID})
	require.NoError(t, err)
	_, err = collectSvc.Collect(domain.Item{ID: 1, Amount: 1}, domain.Collector{ID: detectorUser.ID})
	require.NoError(t, err)

	hasher := img.NewPHasher()
	detector := detectfakes.NewDetector()
	searchSvc := application.NewSearchService(repo, detector, hasher)
	srv.RegisterRoutes(func(r fiber.Router) {
		adapter.SearchRoutes(r.Group("/"), config.Oidc{}, authSvc, searchSvc)
	})

	return srv
}
