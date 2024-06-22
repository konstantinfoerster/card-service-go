package detect_test

import (
	"context"
	"net/http"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/collection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detect"
	detectfakes "github.com/konstantinfoerster/card-service-go/internal/cards/detect/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/cards/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	oidcfakes "github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc/fakes"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/common/web"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var detectorUser = auth.User{ID: "detectorUser", Username: "detectorUser"}
var validUser = auth.User{ID: "myUser", Username: "myUser"}

func TestDetect(t *testing.T) {
	srv := testServer(t)
	fImg, err := os.Open(path.Join(currentDir(), "testdata", "cardImageModified.jpg"))
	defer commonio.Close(fImg)
	require.NoError(t, err)

	req := test.NewRequest(
		test.WithMethod(http.MethodPost),
		test.WithURL("http://localhost/detect"),
		test.WithMultipartFile(t, fImg, fImg.Name()),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	score := 4
	expected := []cards.CardDTO{
		{
			ItemDTO: cards.ItemDTO{ID: 1},
			Name:    "Ancestor's Chosen",
			Image:   "cardImage.jpg",
			Score:   &score,
		},
	}
	body := test.FromJSON[web.PagedResponse[cards.CardDTO]](t, resp)
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.ElementsMatch(t, expected, body.Data)
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}

func TestDetectWithUser(t *testing.T) {
	srv := testServer(t)
	fImg, err := os.Open(path.Join(currentDir(), "testdata", "cardImageModified.jpg"))
	defer commonio.Close(fImg)
	require.NoError(t, err)
	token := test.Base64Encoded(t, &oidc.JSONWebToken{IDToken: detectorUser.ID})
	req := test.NewRequest(
		test.WithMethod(http.MethodPost),
		test.WithURL("http://localhost/detect"),
		test.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
		test.WithMultipartFile(t, fImg, fImg.Name()),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	score := 4
	expected := []cards.CardDTO{
		{
			ItemDTO: cards.ItemDTO{ID: 1, Amount: 1},
			Name:    "Ancestor's Chosen",
			Image:   "cardImage.jpg",
			Score:   &score,
		},
	}
	body := test.FromJSON[web.PagedResponse[cards.CardDTO]](t, resp)
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.ElementsMatch(t, expected, body.Data)
}

func testServer(t *testing.T) *web.Server {
	srv := web.NewHTTPTestServer()

	authSvc := oidcfakes.NewUserService(validUser, detectorUser)
	repo, err := fakes.NewRepository(detect.NewPHasher())
	require.NoError(t, err)
	collectSvc := collection.NewService(repo, repo)

	ctx := context.Background()
	_, err = collectSvc.Collect(ctx, collection.Item{ID: 1, Amount: 1, Owner: detectorUser.ID})
	require.NoError(t, err)

	hasher := detect.NewPHasher()
	detector := detectfakes.NewDetector()
	svc := detect.NewDetectService(repo, detector, hasher)
	srv.RegisterRoutes(func(r fiber.Router) {
		detect.Routes(r.Group("/"), config.Oidc{}, authSvc, svc)
	})

	return srv
}
