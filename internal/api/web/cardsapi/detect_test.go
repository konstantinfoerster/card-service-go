package cardsapi_test

import (
	"context"
	"encoding/base64"
	"io"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/cardsapi"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/imaging"
	"github.com/konstantinfoerster/card-service-go/internal/cards/memory"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDetectRepository struct {
	scores cards.Scores
}

func (s stubDetectRepository) Top5MatchesByHash(_ context.Context, _ ...cards.Hash) (cards.Scores, error) {
	return s.scores, nil
}

func TestDetect(t *testing.T) {
	four := 4
	cases := []struct {
		name     string
		img      string
		user     test.RequestOpt
		scores   cards.Scores
		expected []cardsapi.Card
	}{
		{
			name:   "match",
			img:    "cardImageModified.jpg",
			scores: cards.Scores{{ID: cards.NewID(1).WithFace(1), Score: 4}},
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
			name:   "match with user",
			img:    "cardImageModified.jpg",
			user:   test.WithUser("myuser"),
			scores: cards.Scores{{ID: cards.NewID(1).WithFace(1), Score: 4}},
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
			scores:   cards.Scores{},
			expected: []cardsapi.Card{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := detectTestServer(t, stubDetectRepository{scores: tc.scores})

			fImg, err := os.Open(path.Join(currentDir(), "testdata", tc.img))
			defer aio.Close(fImg)
			require.NoError(t, err)
			rawImg, err := io.ReadAll(fImg)
			require.NoError(t, err)
			req := test.NewRequest(
				t.Context(),
				test.WithMethod(web.MethodPost),
				test.WithURL("http://localhost/detect"),
				tc.user,
				test.WithJSONBody(t, cardsapi.DetectRequest{
					Image: base64.StdEncoding.EncodeToString(rawImg),
				}),
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

func TestDetectInvalidImage(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{
			name: "no image",
			body: cardsapi.DetectRequest{},
		},
		{
			name: "no base64 image",
			body: cardsapi.DetectRequest{
				Image: "not base64",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := detectTestServer(t, stubDetectRepository{})

			req := test.NewRequest(
				t.Context(),
				test.WithMethod(web.MethodPost),
				test.WithURL("http://localhost/detect"),
				test.WithJSONBody(t, tc.body),
			)
			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			assert.Equal(t, web.StatusBadRequest, resp.StatusCode)
		})
	}
}

func detectTestServer(t *testing.T, dRepo cards.DetectRepository) *web.Server {
	seed, err := test.CardSeed()
	require.NoError(t, err)
	item, err := cards.NewCollectable(cards.NewID(1), 1)
	require.NoError(t, err)
	loggedInUser := web.NewUser("myuser")
	collected := map[string][]cards.Collectable{
		loggedInUser.ID: {item},
	}
	cRepo, err := memory.NewCardRepository(seed, collected)
	require.NoError(t, err)

	detector := imaging.NewFakeDetector()
	svc := cards.NewDetectService(cRepo, dRepo, detector)

	srv := web.NewTestServer()
	srv.RegisterRoutes(func(r fiber.Router) {
		cfg := web.Auth{
			HeaderUserID:    web.HeaderUserID,
			HeaderUserEmail: web.HeaderUserEmail,
		}

		cardsapi.DetectRoutes(r.Group("/"), cfg, web.Config{}, svc)
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
