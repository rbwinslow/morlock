package main_test

import (
	"github.com/rbwinslow/morlock/test_util"
	"github.com/rbwinslow/morlock/web"

	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"time"
	"github.com/rbwinslow/morlock/api"
)

var _ = Describe("request handlers", func() {
	Describe("history endpoint", func() {
		It("should return the history for a file", func() {
			// Given
			filename := "foo.txt"
			test_util.WithTemporaryGitRepo(func(repo *test_util.TemporaryGitRepo) {
				repo.MustAddFile(filename, "foo")
				hash1 := repo.MustCommit("foo!")
				repo.MustAddFile(filename, "bar")
				hash2 := repo.MustCommit("bar!")

				filepath := path.Join(repo.Path, filename)
				URL := fmt.Sprintf("http://localhost/history?path=%s", url.QueryEscape(filepath))
				req, err := http.NewRequest("GET", URL, nil)
				if err != nil {
					panic(err)
				}

				w := httptest.NewRecorder()

				// When
				main.HistoryHandler(w, req)

				// Then
				response := w.Result()
				Expect(response.StatusCode).To(Equal(http.StatusOK))
				Expect(response.Header["Content-Type"][0]).To(Equal("application/json"))

				var result []struct {
					Hash   string
					Author string
					Date   time.Time
				}
				body, err := ioutil.ReadAll(response.Body)
				Expect(err).To(BeNil())
				err = json.Unmarshal(body, &result)
				Expect(err).To(BeNil())

				Expect(len(result)).To(Equal(2))
				Expect(api.MustBeHash(result[0].Hash).Short()).To(Equal(hash2))
				Expect(api.MustBeHash(result[1].Hash).Short()).To(Equal(hash1))
				for i := 0; i < 2; i++ {
					Expect(result[i].Author).To(ContainSubstring(repo.UserName))
					Expect(result[i].Date).To(BeTemporally("<", time.Now(), 10 * time.Second))
				}
			})
		})
	})
})
