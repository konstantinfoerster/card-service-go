package cardsapi_test

import (
	"encoding/gob"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	fibermemory "github.com/gofiber/storage/memory/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/cardsapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/imaging"
	"github.com/konstantinfoerster/card-service-go/internal/cards/memory"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	srv := detectTestServer(t)
	four := 4
	cases := []struct {
		name       string
		img        string
		userCookie test.RequestOpt
		expected   []cardsapi.Card
	}{
		{
			name: "match",
			img:  "cardImageModified.jpg",
			expected: []cardsapi.Card{
				{
					ID:    "Y2FyZD0xJmZhY2U9MQ==",
					Name:  "Ancestor's Chosen",
					Image: "cardImage.jpg",
					Set: cardsapi.Set{
						Code: "10E",
						Name: "Tenth Edition",
					},
					Number:     "1",
					Confidence: &four,
				},
			},
		},
		{
			name:       "match with user",
			img:        "cardImageModified.jpg",
			userCookie: test.WithSession("validSessionID"),
			expected: []cardsapi.Card{
				{
					ID:     "Y2FyZD0xJmZhY2U9MQ==",
					Amount: 1,
					Name:   "Ancestor's Chosen",
					Image:  "cardImage.jpg",
					Set: cardsapi.Set{
						Code: "10E",
						Name: "Tenth Edition",
					},
					Number:     "1",
					Confidence: &four,
				},
			},
		},
		{
			name:     "no score",
			img:      "noscore.jpg",
			expected: []cardsapi.Card{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fImg, err := os.Open(path.Join(currentDir(), "testdata", tc.img))
			defer aio.Close(fImg)
			require.NoError(t, err)
			req := test.NewRequest(
				test.WithMethod(web.MethodPost),
				test.WithURL("http://localhost/detect"),
				tc.userCookie,
				test.WithMultipartFile(t, fImg, fImg.Name()),
			)
			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, web.StatusOK, resp.StatusCode)
			body := test.FromJSON[cardsapi.PagedResponse[cardsapi.Card]](t, resp.Body)
			assert.False(t, body.HasMore)
			assert.Equal(t, 1, body.Page)
			assert.ElementsMatch(t, tc.expected, body.Data)
		})
	}
}

func detectTestServer(t *testing.T) *web.Server {
	srv := web.NewTestServer()

	cfg := postgres.Images{Host: "testdata"}
	seed, err := test.CardSeed()
	require.NoError(t, err)
	dRepo, err := memory.NewDetectRepository(seed, cfg)
	require.NoError(t, err)
	item, err := cards.NewCollectable(cards.NewID(1), 1)
	require.NoError(t, err)
	loggedInUser := auth.NewUser("myuser")
	collected := map[string][]cards.Collectable{
		loggedInUser.ID: {item},
	}
	cRepo, err := memory.NewCardRepository(seed, collected)
	require.NoError(t, err)

	detector := imaging.NewFakeDetector()
	svc := cards.NewDetectService(cRepo, dRepo, detector)

	gob.Register(auth.User{})
	sCfg := session.Config{
		KeyLookup: "cookie:SESSION",
		Storage:   fibermemory.New(),
	}
	err = sCfg.Storage.Set("validSessionID", test.AsSessionData(t, auth.UserContextKey, loggedInUser), 0)
	require.NoError(t, err)

	store := session.New(sCfg)
	srv.RegisterRoutes(func(r fiber.Router) {
		cardsapi.DetectRoutes(r.Group("/"), web.NewAuthMiddleware(store), svc)
	})

	return srv
}

func currentDir() string {
	_, cf, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current dir")
	}

	return path.Join(path.Dir(cf))
}
