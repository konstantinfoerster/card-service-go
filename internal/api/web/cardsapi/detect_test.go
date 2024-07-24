package cardsapi_test

import (
	"os"
	"path"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/cardsapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/memory"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/image"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	srv, _ := detectTestServer(t)
	fImg, err := os.Open(path.Join(currentDir(), "testdata", "cardImageModified.jpg"))
	defer aio.Close(fImg)
	require.NoError(t, err)

	req := test.NewRequest(
		test.WithMethod(web.MethodPost),
		test.WithURL("http://localhost/detect"),
		test.WithMultipartFile(t, fImg, fImg.Name()),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, web.StatusOK, resp.StatusCode)
	score := 4
	expected := []cardsapi.Card{
		{
			Item:  cardsapi.Item{ID: 1},
			Name:  "Ancestor's Chosen",
			Image: "cardImage.jpg",
			Score: &score,
		},
	}
	body := test.FromJSON[cardsapi.PagedResponse[cardsapi.Card]](t, resp.Body)
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.ElementsMatch(t, expected, body.Data)
}

func TestDetectWithUser(t *testing.T) {
	srv, provider := detectTestServer(t)
	fImg, err := os.Open(path.Join(currentDir(), "testdata", "cardImageModified.jpg"))
	defer aio.Close(fImg)
	require.NoError(t, err)
	token := provider.Token("myuser")
	req := test.NewRequest(
		test.WithMethod(web.MethodPost),
		test.WithURL("http://localhost/detect"),
		test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token)),
		test.WithMultipartFile(t, fImg, fImg.Name()),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, web.StatusOK, resp.StatusCode)
	score := 4
	expected := []cardsapi.Card{
		{
			Item:  cardsapi.Item{ID: 1, Amount: 1},
			Name:  "Ancestor's Chosen",
			Image: "cardImage.jpg",
			Score: &score,
		},
	}
	body := test.FromJSON[cardsapi.PagedResponse[cardsapi.Card]](t, resp.Body)
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.ElementsMatch(t, expected, body.Data)
}

func detectTestServer(t *testing.T) (*web.Server, *auth.FakeProvider) {
	srv := web.NewHTTPTestServer()

	cfg := config.Images{Host: "testdata"}
	hasher := image.NewPHasher()
	item, err := cards.NewCollectable(1, 1)
	require.NoError(t, err)
	collected := map[string][]cards.Collectable{
		validClaim.ID: {item},
	}
	seed, err := test.CardSeed()
	require.NoError(t, err)
	repo, err := memory.NewDetectRepository(
		seed, collected, cfg, hasher,
	)
	require.NoError(t, err)

	oCfg := config.Oidc{}
	provider := auth.NewFakeProvider(auth.WithClaims(validClaim))
	authSvc := auth.New(oCfg, provider)
	detector := image.NewFakeDetector()
	svc := cards.NewDetectService(repo, detector, hasher)
	srv.RegisterRoutes(func(r fiber.Router) {
		cardsapi.DetectRoutes(r.Group("/"), web.NewAuthMiddleware(oCfg, authSvc), svc)
	})

	return srv, provider
}
