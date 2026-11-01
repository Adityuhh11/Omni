package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"
)

// Relay carries MCP messages between an AI app (one JSON-RPC message per line on in and out, the protocol's stdio transport)
// and the running Omni, whose address it reads from file before each message, so an Omni started again is found. It returns
// when in ends.
func Relay(in io.Reader, out io.Writer, file string) error {
	client := &http.Client{Timeout: 2 * time.Minute}
	lines := bufio.NewScanner(in)
	lines.Buffer(make([]byte, 64<<10), 4<<20)
	for lines.Scan() {
		line := bytes.TrimSpace(lines.Bytes())
		if len(line) == 0 {
			continue
		}
		answer, err := forward(client, file, line)
		if err != nil {
			answer = failure(line, err)
		}
		if len(answer) > 0 {
			if _, err := fmt.Fprintf(out, "%s\n", bytes.TrimSpace(answer)); err != nil {
				return err
			}
		}
	}
	return lines.Err()
}

func forward(client *http.Client, file string, message []byte) ([]byte, error) {
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("Omni is not running, or “Use in other AI apps” is off in Settings, Omni AI")
	}
	if err != nil {
		return nil, fmt.Errorf("read Omni's MCP address: %w", err)
	}
	var addr Address
	if err := json.Unmarshal(data, &addr); err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	req, err := http.NewRequest(http.MethodPost, addr.URL, bytes.NewReader(message))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+addr.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Omni did not answer; is it running? (%v)", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Omni answered %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}
	return body, nil
}

// failure is the error answer to a request that could not be relayed; a notification gets none.
func failure(message []byte, err error) []byte {
	var req struct {
		ID json.RawMessage `json:"id"`
	}
	if json.Unmarshal(message, &req) != nil || len(req.ID) == 0 {
		return nil
	}
	b, _ := json.Marshal(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{-32000, err.Error()}})
	return b
}
