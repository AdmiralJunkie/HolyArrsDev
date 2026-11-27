package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/HolyArrs/holyarrs/app/backend/internal/core/database"
	domain "github.com/HolyArrs/holyarrs/app/backend/internal/core/model"
	"github.com/HolyArrs/holyarrs/app/backend/internal/core/secrets"
)

// A person's Jellyfin sign-in for the built-in player (HolyArrsDev fork).
// When a person links their Jellyfin account in My account and asks to keep
// it for playback, the token Jellyfin gives is kept sealed in the vault on
// their Jellyfin link (user_links.token_secret_id, as Plex tokens are), so
// the player can ask Jellyfin for streams as that person: their resume
// points, watched marks and library limits. It never leaves the server.
// Whatever removes the link removes the token in the same write
// (deleteMediaLink, endOutsideLinks).

const (
	playerTokenKind  = "jellyfin_player_token"
	playerDeviceName = product + " player"
)

// playerDeviceID is the Jellyfin device id a person's player sign-in is
// made as: one per person, never HolyArrs's own id (see checkMediaServerAs).
// Later Jellyfin calls with the kept token must send the same id.
func (a *Auth) playerDeviceID(ctx context.Context, userID int64) string {
	return a.clientID(ctx) + "-player-" + strconv.FormatInt(userID, 10)
}

// PlayerSignIn is a person's kept Jellyfin sign-in.
type PlayerSignIn struct {
	AccountID string // the person's Jellyfin user id
	Token     string
	DeviceID  string
	// Device is the device name the sign-in was made with; send it with
	// DeviceID in the Authorization header.
	Device string
}

// JellyfinPlayer is the Jellyfin sign-in kept for userID's player. ok is
// false when none is kept, the person has no Jellyfin link, or they are
// turned off.
func (a *Auth) JellyfinPlayer(ctx context.Context, userID int64) (PlayerSignIn, bool, error) {
	if a.vault == nil {
		return PlayerSignIn{}, false, nil
	}
	var ext, state string
	var ref sql.NullInt64
	err := a.s.Read().QueryRowContext(ctx, `SELECT l.external_id, l.token_secret_id, u.state FROM user_links l JOIN users u ON u.id = l.user_id
		WHERE l.provider = 'jellyfin' AND l.user_id = ?`, userID).Scan(&ext, &ref, &state)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return PlayerSignIn{}, false, nil
	case err != nil:
		return PlayerSignIn{}, false, err
	case !ref.Valid || state != "active":
		return PlayerSignIn{}, false, nil
	}
	tok, err := a.vault.Get(ctx, ref.Int64)
	if err != nil || tok == "" {
		return PlayerSignIn{}, false, err
	}
	return PlayerSignIn{AccountID: ext, Token: tok, DeviceID: a.playerDeviceID(ctx, userID), Device: playerDeviceName}, true, nil
}

// keepPlayerTokenTx makes the player sign-in kept on a person's link exactly
// tok: the one kept before goes, and "" keeps none. A link that is not there
// keeps nothing (sql.ErrNoRows is not returned: linkMediaTx just made it).
func (a *Auth) keepPlayerTokenTx(ctx context.Context, tx *sql.Tx, userID int64, kind, extID, tok string) error {
	ids, err := tokenIDs(ctx, tx, "SELECT token_secret_id FROM user_links WHERE token_secret_id IS NOT NULL AND user_id = ? AND provider = ? AND external_id = ?",
		userID, kind, extID)
	if err != nil {
		return err
	}
	if err := secrets.DeleteTx(ctx, tx, ids...); err != nil {
		return err
	}
	var ref any
	if tok != "" && a.vault != nil {
		id, err := a.vault.PutTx(tx, playerTokenKind, tok, domain.System)
		if err != nil {
			return err
		}
		ref = id
	}
	if _, err := tx.ExecContext(ctx, "UPDATE user_links SET token_secret_id = ? WHERE user_id = ? AND provider = ? AND external_id = ?",
		ref, userID, kind, extID); err != nil {
		return err
	}
	if kind != ProviderJellyfin {
		return nil
	}
	return a.clearPlayerLostTx(tx, userID)
}

// playerTokenOf is the player sign-in kept on userID's Jellyfin link
// whatever their state; "" when none is kept or it cannot be read.
func (a *Auth) playerTokenOf(ctx context.Context, userID int64) string {
	if a.vault == nil {
		return ""
	}
	var ref sql.NullInt64
	err := a.s.Read().QueryRowContext(ctx, "SELECT token_secret_id FROM user_links WHERE provider = 'jellyfin' AND user_id = ?", userID).Scan(&ref)
	if err != nil || !ref.Valid {
		return ""
	}
	tok, err := a.vault.Get(ctx, ref.Int64)
	if err != nil {
		return ""
	}
	return tok
}

// endDroppedPlayerToken signs before out of Jellyfin when a change took it
// off userID's link (unlinked, linked again without keeping, another
// account, the person deleted), so its device leaves Jellyfin's Devices
// list. Read before with playerTokenOf ahead of the change; a change that
// failed left it kept and nothing happens.
func (a *Auth) endDroppedPlayerToken(ctx context.Context, userID int64, before string) {
	if before == "" || a.playerTokenOf(ctx, userID) == before {
		return
	}
	ms, err := a.linkServers(ctx)
	if err != nil {
		log.Printf("auth: ending the player sign-in of user %d: %v", userID, err)
	}
	for _, m := range ms {
		if m.Kind == ProviderJellyfin {
			a.endPlayerToken(ctx, m.URL, userID, before)
		}
	}
}

// endPlayerToken ends a player sign-in on the server; failures are only
// logged.
func (a *Auth) endPlayerToken(ctx context.Context, serverURL string, userID int64, tok string) {
	authz := fmt.Sprintf(`MediaBrowser Client="%s", Device="%s", DeviceId="%s", Version="1", Token=%q`,
		product, playerDeviceName, a.playerDeviceID(ctx, userID), tok)
	hdr := map[string]string{"Authorization": authz, "X-Emby-Token": tok}
	if _, err := a.serverFetch(context.WithoutCancel(ctx), http.MethodPost, serverURL+"/Sessions/Logout", hdr, nil, nil); err != nil {
		log.Printf("auth: ending the player sign-in of user %d: %v", userID, err)
	}
}

// playerLostKey is the this_app setting that remembers a person's player
// sign-in went with a removed Jellyfin server: "1" until they link again
// (or unlink), "" otherwise. My account and the bell read it.
func playerLostKey(userID int64) string { return "player_lost." + strconv.FormatInt(userID, 10) }

// playerLost says userID's player sign-in went with a removed server.
func (a *Auth) playerLost(ctx context.Context, userID int64) bool {
	return database.SettingOr(ctx, a.s.Read(), settingsGroup, playerLostKey(userID), "") == "1"
}

// PlayerServerRemoving runs just before the owner removes a media server,
// while its address is still known. When it is the last connected Jellyfin
// server, every kept player sign-in is signed out there (so the devices leave
// Jellyfin), deleted here, and its person is told (playerLost). The links
// themselves stay: watched marks come back when the server is connected again.
func (a *Auth) PlayerServerRemoving(ctx context.Context, kind, serverURL string) {
	if kind != ProviderJellyfin || a.vault == nil {
		return
	}
	ms, err := a.linkServers(ctx)
	if err != nil {
		log.Printf("auth: player sign-ins on a removed server: %v", err)
	}
	for _, m := range ms {
		if m.Kind == ProviderJellyfin && !sameServer(m.URL, serverURL) {
			return // another Jellyfin server stays connected
		}
	}
	rows, err := a.s.Read().QueryContext(ctx, "SELECT user_id, token_secret_id FROM user_links WHERE provider = 'jellyfin' AND token_secret_id IS NOT NULL")
	if err != nil {
		log.Printf("auth: player sign-ins on a removed server: %v", err)
		return
	}
	type kept struct{ user, ref int64 }
	var all []kept
	for rows.Next() {
		var k kept
		if err := rows.Scan(&k.user, &k.ref); err == nil {
			all = append(all, k)
		}
	}
	_ = rows.Close()
	if len(all) == 0 {
		return
	}
	for _, k := range all {
		if tok, err := a.vault.Get(ctx, k.ref); err == nil && tok != "" {
			a.endPlayerToken(ctx, serverURL, k.user, tok)
		}
	}
	err = a.s.WriteCore(ctx, func(tx *sql.Tx) error {
		for _, k := range all {
			if err := secrets.DeleteTx(ctx, tx, k.ref); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE user_links SET token_secret_id = NULL WHERE provider = 'jellyfin' AND user_id = ? AND token_secret_id = ?", k.user, k.ref); err != nil {
				return err
			}
			if err := database.PutSetting(tx, settingsGroup, playerLostKey(k.user), "1", domain.System, a.now()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("auth: player sign-ins on a removed server: %v", err)
		return
	}
	log.Printf("auth: the Jellyfin server was removed; ended %d player sign-in(s)", len(all))
}

// clearPlayerLostTx forgets that userID's player sign-in went with a
// removed server: they linked again or unlinked.
func (a *Auth) clearPlayerLostTx(tx *sql.Tx, userID int64) error {
	if database.SettingOr(context.Background(), tx, settingsGroup, playerLostKey(userID), "") == "" {
		return nil
	}
	return database.PutSetting(tx, settingsGroup, playerLostKey(userID), "", domain.System, a.now())
}
