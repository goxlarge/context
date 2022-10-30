package cocktaildb

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/goxlarge/context/userip"
)

// Results is an ordered list of search results.
type Recipes struct {
	Drinks Drinks `json:"drinks"`
}
type Drinks []Drink

type Drink struct {
	StrDrink        string `json:"strDrink"`
	StrDrinkThumb string `json:"strDrinkThumb"`
}

func Search(ctx context.Context, query string) (Recipes, error) {
	req, err := http.NewRequest("GET", "https://www.thecocktaildb.com/api/json/v1/1/search.php?", nil)
	if err != nil {
		return Recipes{}, err
	}
	q := req.URL.Query()
	q.Set("s", query)

	// If ctx is carrying the user IP address, forward it to the server.
	// Google APIs use the user IP to distinguish server-initiated requests
	// from end-user requests.
	if userIP, ok := userip.FromContext(ctx); ok {
		q.Set("userip", userIP.String())
	}
	req.URL.RawQuery = q.Encode()

	// Issue the HTTP request and handle the response. The httpDo function
	// cancels the request if ctx.Done is closed.
	var results Recipes
	err = httpDo(ctx, req, func(resp *http.Response, err error) error {
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			return err
		}
		return nil
	})
	// httpDo waits for the closure we provided to return, so it's safe to
	// read results here.
	return results, err
}

// httpDo issues the HTTP request and calls f with the response. If ctx.Done is
// closed while the request or f is running, httpDo cancels the request, waits
// for f to exit, and returns ctx.Err. Otherwise, httpDo returns f's error.
func httpDo(ctx context.Context, req *http.Request, f func(*http.Response, error) error) error {
	// Run the HTTP request in a goroutine and pass the response to f.
	c := make(chan error, 1)
	req = req.WithContext(ctx)
	go func() { c <- f(http.DefaultClient.Do(req)) }()
	select {
	case <-ctx.Done():
		<-c // Wait for f to return.
		return ctx.Err()
	case err := <-c:
		return err
	}
}
