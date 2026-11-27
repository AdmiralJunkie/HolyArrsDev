package accounts

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakePlayerJellyfin answers AuthenticateByName like Jellyfin, giving a
// different token to the player's device than to HolyArrs's own, and counts
// ended sign-ins by token.
type fakePlayerJellyfin struct {
	mu      sync.Mutex
	devices []string       // the DeviceId of each sign-in, in order
	ended   map[string]int // token -> times ended
}

func (j *fakePlayerJellyfin) serve(t *testing.T) *httptest.Server {
	j.ended = map[string]int{}
	m := http.NewServeMux()
	m.HandleFunc("POST /Users/AuthenticateByName", func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		_, dev, _ := strings.Cut(authz, `DeviceId="`)
		dev, _, _ = strings.Cut(dev, `"`)
		j.mu.Lock()
		j.devices = append(j.devices, dev)
		j.mu.Unlock()
		var b struct{ Username, Pw string }
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Username != "mum" || b.Pw != "right" {
			http.Error(w, "Error processing request.", 401)
			return
		}
		tok := "tok"
		if strings.Contains(dev, "-player-") {
			tok = "playtok"
		}
		_, _ = w.Write([]byte(`{"User":{"Id":"abc123","Name":"Mum","Policy":{"IsDisabled":false}},"AccessToken":"` + tok + `"}`))
	})
	m.HandleFunc("POST /Sessions/Logout", func(w http.ResponseWriter, r *http.Request) {
		j.mu.Lock()
		j.ended[r.Header.Get("X-Emby-Token")]++
		j.mu.Unlock()
		w.WriteHeader(204)
	})
	s := httptest.NewServer(m)
	t.Cleanup(s.Close)
	return s
}

func (j *fakePlayerJellyfin) lastDevice() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	if len(j.devices) == 0 {
		return ""
	}
	return j.devices[len(j.devices)-1]
}

func (j *fakePlayerJellyfin) endedCount(tok string) int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.ended[tok]
}

func setPlayerState(t *testing.T, f *fixture, uid int64, state string) {
	t.Helper()
	err := f.s.WriteCore(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE users SET state = ? WHERE id = ?", state, uid)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func playerTokens(t *testing.T, f *fixture) int {
	t.Helper()
	var n int
	if err := f.s.Read().QueryRow("SELECT count(*) FROM secrets WHERE kind = ?", playerTokenKind).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Linking Jellyfin from My account with keep_for_playback keeps the token,
// made as the person's own player device; linking again without it, or
// unlinking, drops it and signs it out of Jellyfin.
func TestJellyfinPlayerSignIn(t *testing.T) {
	f := gated(t, Options{})
	j := &fakePlayerJellyfin{}
	srv := j.serve(t)
	f.a.SetMediaLinks(MediaLinks{
		Servers: func(context.Context) ([]LinkServer, error) {
			return []LinkServer{{ID: 7, Name: "Den", Kind: "jellyfin", URL: srv.URL}}, nil
		},
		Accounts: func(context.Context, int64) ([]ServerAccount, error) { return nil, nil },
	})
	own := ownerOf(t, f)
	ana := join(t, f, invite(t, f, own, `{"role":"member"}`), "ana")
	uid := userID(t, f, "ana")
	ctx := context.Background()
	wantDevice := "-player-" + strconv.FormatInt(uid, 10)

	// Kept.
	resp, out := ana.do(t, f, "POST", "/api/v1/auth/media-link/7", `{"name":"mum","password":"right","keep_for_playback":true}`)
	if resp.StatusCode != 200 || !strings.Contains(out, `"kept":true`) {
		t.Fatalf("link and keep: %d %s", resp.StatusCode, out)
	}
	if d := j.lastDevice(); !strings.HasSuffix(d, wantDevice) {
		t.Fatalf("signed in as device %q, want one ending %q", d, wantDevice)
	}
	if j.endedCount("playtok") != 0 {
		t.Fatal("the kept sign-in was ended")
	}
	in, ok, err := f.a.JellyfinPlayer(ctx, uid)
	if err != nil || !ok || in.Token != "playtok" || in.AccountID != "abc123" || !strings.HasSuffix(in.DeviceID, wantDevice) || in.Device != playerDeviceName {
		t.Fatalf("player sign-in = %+v %t %v", in, ok, err)
	}
	if _, out := ana.do(t, f, "GET", "/api/v1/auth/media-link", ""); !strings.Contains(out, `"kept":true`) || strings.Contains(out, "playtok") {
		t.Fatalf("my links: %s", out)
	}
	if n := playerTokens(t, f); n != 1 {
		t.Fatalf("%d player tokens kept, want 1", n)
	}

	// Linking the same account again without the flag drops the token.
	resp, out = ana.do(t, f, "POST", "/api/v1/auth/media-link/7", `{"name":"mum","password":"right"}`)
	if resp.StatusCode != 200 || strings.Contains(out, `"kept"`) {
		t.Fatalf("link without keeping: %d %s", resp.StatusCode, out)
	}
	if d := j.lastDevice(); strings.Contains(d, "-player-") {
		t.Fatalf("a plain link used the player device %q", d)
	}
	if j.endedCount("tok") != 1 {
		t.Fatal("a plain link's sign-in was not ended")
	}
	if j.endedCount("playtok") != 1 {
		t.Fatalf("the dropped player sign-in was ended %d times, want 1", j.endedCount("playtok"))
	}
	if _, ok, _ := f.a.JellyfinPlayer(ctx, uid); ok || playerTokens(t, f) != 0 {
		t.Fatal("the player sign-in is still kept")
	}

	// Kept again, then a turned-off person has none.
	if resp, out := ana.do(t, f, "POST", "/api/v1/auth/media-link/7", `{"name":"mum","password":"right","keep_for_playback":true}`); resp.StatusCode != 200 {
		t.Fatalf("keep again: %d %s", resp.StatusCode, out)
	}
	setPlayerState(t, f, uid, "disabled")
	if _, ok, _ := f.a.JellyfinPlayer(ctx, uid); ok {
		t.Fatal("a turned-off person has a player sign-in")
	}
	setPlayerState(t, f, uid, "active")

	// Unlinking drops it.
	if resp, _ := ana.do(t, f, "DELETE", "/api/v1/auth/media-link/jellyfin", ""); resp.StatusCode != 204 {
		t.Fatalf("unlink: %d", resp.StatusCode)
	}
	if n := playerTokens(t, f); n != 0 {
		t.Fatalf("%d player tokens left after unlinking", n)
	}
	if j.endedCount("playtok") != 2 {
		t.Fatalf("unlinking did not end the player sign-in on Jellyfin (%d ends)", j.endedCount("playtok"))
	}
}

// Deleting a person signs their kept player sign-in out of Jellyfin.
func TestDeletingPersonEndsPlayerSignIn(t *testing.T) {
	f := gated(t, Options{})
	j := &fakePlayerJellyfin{}
	srv := j.serve(t)
	f.a.SetMediaLinks(MediaLinks{
		Servers: func(context.Context) ([]LinkServer, error) {
			return []LinkServer{{ID: 7, Name: "Den", Kind: "jellyfin", URL: srv.URL}}, nil
		},
		Accounts: func(context.Context, int64) ([]ServerAccount, error) { return nil, nil },
	})
	own := ownerOf(t, f)
	ana := join(t, f, invite(t, f, own, `{"role":"member"}`), "ana")
	if resp, out := ana.do(t, f, "POST", "/api/v1/auth/media-link/7", `{"name":"mum","password":"right","keep_for_playback":true}`); resp.StatusCode != 200 {
		t.Fatalf("link and keep: %d %s", resp.StatusCode, out)
	}
	id := strconv.FormatInt(userID(t, f, "ana"), 10)
	if resp, out := own.do(t, f, "DELETE", "/api/v1/users/"+id, ""); resp.StatusCode != 204 {
		t.Fatalf("delete: %d %s", resp.StatusCode, out)
	}
	if n := playerTokens(t, f); n != 0 || j.endedCount("playtok") != 1 {
		t.Fatalf("after deleting: %d tokens kept, %d ends", n, j.endedCount("playtok"))
	}
}

// Removing the last Jellyfin server signs every kept player sign-in out of it,
// deletes them, keeps the links, and tells each person until they link again.
func TestRemovingJellyfinEndsPlayerSignIns(t *testing.T) {
	f := gated(t, Options{})
	j := &fakePlayerJellyfin{}
	srv := j.serve(t)
	f.a.SetMediaLinks(MediaLinks{
		Servers: func(context.Context) ([]LinkServer, error) {
			return []LinkServer{{ID: 7, Name: "Den", Kind: "jellyfin", URL: srv.URL}}, nil
		},
		Accounts: func(context.Context, int64) ([]ServerAccount, error) { return nil, nil },
	})
	own := ownerOf(t, f)
	ana := join(t, f, invite(t, f, own, `{"role":"member"}`), "ana")
	uid := userID(t, f, "ana")
	ctx := context.Background()
	if _, out := ana.do(t, f, "GET", "/api/v1/auth/media-link", ""); !strings.Contains(out, `"player_lost":false`) {
		t.Fatalf("before: %s", out)
	}
	if resp, out := ana.do(t, f, "POST", "/api/v1/auth/media-link/7", `{"name":"mum","password":"right","keep_for_playback":true}`); resp.StatusCode != 200 {
		t.Fatalf("link and keep: %d %s", resp.StatusCode, out)
	}

	// An Emby server going changes nothing.
	f.a.PlayerServerRemoving(ctx, "emby", "http://emby.example")
	if playerTokens(t, f) != 1 {
		t.Fatal("removing an Emby server dropped the Jellyfin player sign-in")
	}

	f.a.PlayerServerRemoving(ctx, "jellyfin", srv.URL+"/")
	if n := playerTokens(t, f); n != 0 || j.endedCount("playtok") != 1 {
		t.Fatalf("after removing: %d tokens kept, %d ends", n, j.endedCount("playtok"))
	}
	links, _ := LinksOfPerson(ctx, f.s.Read(), uid)
	if len(links) != 1 || links[0].ExtID != "abc123" {
		t.Fatalf("the link should stay: %v", links)
	}
	if _, out := ana.do(t, f, "GET", "/api/v1/auth/media-link", ""); !strings.Contains(out, `"player_lost":true`) || strings.Contains(out, `"kept":true`) {
		t.Fatalf("told: %s", out)
	}

	// Linking again (the server is back) ends the alert.
	if resp, out := ana.do(t, f, "POST", "/api/v1/auth/media-link/7", `{"name":"mum","password":"right","keep_for_playback":true}`); resp.StatusCode != 200 {
		t.Fatalf("link again: %d %s", resp.StatusCode, out)
	}
	if _, out := ana.do(t, f, "GET", "/api/v1/auth/media-link", ""); !strings.Contains(out, `"player_lost":false`) || !strings.Contains(out, `"kept":true`) {
		t.Fatalf("after linking again: %s", out)
	}
}
