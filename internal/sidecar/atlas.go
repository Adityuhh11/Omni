package sidecar

import (
	"context"
	"fmt"
	"net/http"
)

// MapStatus is how far the map of the index is (GET /map/status).
type MapStatus struct {
	State   string `json:"state"` // none, building, ready, failed, missing (the map's extra is not installed)
	Message string `json:"message"`
	Count   int    `json:"count"`
	Stale   bool   `json:"stale"`
}

// MapPoints is the whole map: point i is at Coords[12i:12i+12] (three little-endian float32) and belongs to
// Roots[Source[2i:2i+2]] (a little-endian uint16). The engine sends both as base64, which is what []byte decodes from.
type MapPoints struct {
	Coords []byte   `json:"coords"`
	Source []byte   `json:"source"`
	Roots  []string `json:"roots"`
}

// MapPoint is the chunk behind one point.
type MapPoint struct {
	URI     string `json:"uri"`
	Root    string `json:"root"`
	Kind    string `json:"kind"`
	Locator string `json:"locator"`
	Text    string `json:"text"`
}

// Located is where a query lands on the map and the points it was placed among, best first.
type Located struct {
	At     []float64 `json:"at"`
	Points []int     `json:"points"`
}

// MapStatus reports the map; it answers before the model is ready.
func (c *Client) MapStatus(ctx context.Context) (MapStatus, error) {
	var out MapStatus
	err := c.do(ctx, http.MethodGet, "/map/status", nil, &out)
	return out, err
}

// BuildMap starts laying the index out again (a minute or so); a build under way is not started twice.
func (c *Client) BuildMap(ctx context.Context) (MapStatus, error) {
	var out MapStatus
	err := c.do(ctx, http.MethodPost, "/map/build", nil, &out)
	return out, err
}

// Map returns every point of the last built map.
func (c *Client) Map(ctx context.Context) (MapPoints, error) {
	var out MapPoints
	err := c.do(ctx, http.MethodGet, "/map", nil, &out)
	return out, err
}

// MapPoint returns the chunk behind point index.
func (c *Client) MapPoint(ctx context.Context, index int) (MapPoint, error) {
	var out MapPoint
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/map/points/%d", index), nil, &out)
	return out, err
}

// Locate finds where query lands on the map.
func (c *Client) Locate(ctx context.Context, query string) (Located, error) {
	var out Located
	err := c.do(ctx, http.MethodPost, "/map/locate", map[string]string{"query": query}, &out)
	return out, err
}
