# Network: self-hosted realm for LAN and TCP/IP play

`cmd/od2server` runs a realm/lobby for LAN and TCP/IP play. It contacts no
Blizzard service and needs no game files. The implementation is
`d2networking/d2realm`; the packet framing and ids come from
`d2networking/d2gs`.

Legend used below:

- **d2gs**: a packet that already exists in `d2networking/d2gs` (ids and sizes
  marked verified/unverified there). The realm reuses them unchanged.
- **EXTENSION**: OUR design. Nothing like it is known to exist in the original
  protocol (the original kept these functions on Battle.net), and nothing in
  it is verified against the original binary.

## Running

    go build -o od2server ./cmd/od2server
    ./od2server -listen :4000 -saves ./saves -tables ~/git/d2-tables

`-tables` (or `D2_TABLES`) is a folder with `itemstatcost.bin` (or
`ItemStatCost.txt`), `armor.txt`, `weapons.txt`, `misc.txt`, `ItemTypes.txt`,
extracted from the player's own install. Without it the server still checks
headers and core stats, but cannot decode or check items.

## Transport (d2gs framing)

TCP. Each direction is a sequence of blobs: a run of whole packets,
Huffman-compressed (`d2gs.EncodeBlob` / `d2gs.ReadBlob`), prefixed by the
compressed length (1 byte if < 0xF0, else 2 bytes with high nibble 0xF).

Client to server packets are split with `d2gs.SplitClient`, server to client
packets with `d2gs.SplitServer`. Packets used:

| dir | id | meaning | source |
|-----|----|---------|--------|
| C2S | 0x15 | chat: type, 0, message NUL, recipient NUL (recipient empty = everybody) | d2gs (recipient field unverified) |
| C2S | 0x69 | leave the current game (back to the lobby) | d2gs |
| C2S | 0x6c | tunnel: carries the realm messages below | d2gs `Tunnel` |
| S2C | 0xAE | tunnel: carries the realm messages below | d2gs `Tunnel` |
| S2C | 0x26 | chat message | d2gs `ChatMessage` |
| S2C | 0x01 | game flags (difficulty, hardcore, expansion) | d2gs `GameFlags` |
| S2C | 0x03 | load act; `Seed` is the game seed | d2gs `LoadAct` |
| S2C | 0x59 | a player is in the game | d2gs `PlayerInGame` |
| S2C | 0x5c | a player left the game | d2gs `PlayerLeave` |

Gameplay packets are handled by the game simulation of the realm (see
"Gameplay" below); the realm does not relay them between clients.

### Tunnel

Realm messages are `[type][body]` carried by `d2gs.Tunnel`, which splits them
into chunks of at most 0x1f0 bytes and reassembles them with
`d2gs.TunnelAssembler`. Uploads are limited to 64 KiB.

Body encoding (EXTENSION): little endian integers; strings are a length byte
plus bytes (max 255); byte blobs are a u32 length plus bytes; lists are a u16
count plus items. A message with missing or trailing bytes is refused with
`Result{CodeBadRequest}`.

## Session

1. `Hello{version, account}` (type 0x80). The account name (2 to 15 letters,
   digits, `-`, `_`; unique online, case-insensitive) is the lobby name and
   owns the saves. There is no password: this is a LAN realm and the account
   name is not authenticated (EXTENSION; a real login is future work).
   Reply `HelloAck{code, message, roster}`; `message` is the server name,
   `roster` the players in the lobby.
2. Upload or select a character (below).
3. Create or join a game, play, leave (0x69) or disconnect.

Any request before `Hello` is answered with `Result{CodeNoHello}`.

## Messages (EXTENSION)

Client to server (carried in 0x6c):

| type | name | body |
|------|------|------|
| 0x80 | Hello | u8 version (=1), str account |
| 0x81 | ListGames | empty |
| 0x82 | CreateGame | str name, str password, str description, u8 difficulty, u8 maxPlayers, u8 minLevel, u8 maxLevel |
| 0x83 | JoinGame | str name, str password |
| 0x84 | UploadChar | blob .d2s |
| 0x85 | ListChars | empty |
| 0x86 | SelectChar | str name |
| 0x87 | LevelChange | u8 act, u16 level (legacy: the simulation now moves heroes itself) |
| 0x88 | Command | blob d2mp.Command (party, trade, waypoint, respawn) |

Server to client (carried in 0xAE):

| type | name | body |
|------|------|------|
| 0x90 | HelloAck | u8 code, str message, u16 n, n x str |
| 0x91 | Result | u8 op (request type, or 0x69), u8 code, str message |
| 0x92 | GameList | u16 n, n x GameInfo |
| 0x93 | CharList | u16 n, n x (str name, u8 class, u8 level) |
| 0x94 | CharData | str name, blob .d2s |
| 0x95 | Presence | u8 joined, str name (lobby arrival/departure) |
| 0x96 | PlayerLevel | u32 unit id, u8 act, u16 level |
| 0x97 | GameJoined | GameInfo, u32 unit id, u32 game seed |
| 0x98 | World | blob: u16 count + d2mp events (first is a Tick with the server clock) |

`GameInfo`: str name, str description, str creator, u8 difficulty (0 normal,
1 nightmare, 2 hell), u8 players, u8 maxPlayers, u8 minLevel, u8 maxLevel
(0 = no limit), u8 hardcore, u8 expansion, u8 hasPassword.

Result codes: 0 ok, 1 bad request, 2 name taken, 3 no hello, 4 no character,
5 game not found, 6 game exists, 7 game full, 8 bad password, 9 difficulty
locked, 10 level too low, 11 level too high, 12 character rejected,
13 already in a game, 14 not in a game, 15 server full, 16 hardcore/expansion
mismatch, 17 internal error.

## Lobby

Players that said `Hello` and are not in a game are in the lobby. Arrivals and
departures are pushed as `Presence`; a player who returns from a game gets one
`Presence{joined}` per lobby member as a fresh roster. Game list changes are
not pushed: ask with `ListGames`.

Chat (0x15 / 0x26) is scoped by where the sender is: in the lobby it goes to
the lobby (speaker name = account), in a game it goes to that game (speaker =
character name, unit id set). Normal chat is echoed to everybody in scope,
including the sender. A recipient makes it a whisper to that one player in
the same scope (case-insensitive); an unknown recipient gets a system line.
Chat types in 0x26: 1 normal (d2gs), 2 whisper and 4 system are EXTENSION
values. Control characters are stripped; blank messages are dropped.

## Games

Fields follow the original TCP/IP create/join screens:

- name: 1 to 15 printable ASCII characters, unique (case-insensitive).
- password: up to 15 characters, empty = open.
- description: up to 31 characters.
- difficulty: normal, nightmare or hell.
- maxPlayers: 1 to 8 (0 means 8).
- minLevel / maxLevel: EXTENSION, 0 = no limit; max must not be below min.

(The limits are what the realm enforces; they are unverified against the
original screens.)

Creating a game also joins it. The game's hardcore and expansion flags are
the creator's. Rules checked on create and join, in this order of codes:

1. password (join): `BadPassword`
2. full (join): `GameFull`
3. dead hardcore character: `CharInvalid`
4. hardcore vs softcore, classic vs expansion must match: `ModeMismatch`
5. difficulty: a character may play every difficulty up to the active one of
   its save (`Header.ActiveDifficulty`); a never-played character only Normal:
   `DifficultyLocked` (EXTENSION policy, not the original rule)
6. level window: `LevelTooLow` / `LevelTooHigh`

A character can be in one game at a time, and one character name can be
online once. A game ends when its last player leaves. The server holds at most
`-max-games` games and `-max-clients` clients.

### Seed and level sync

The server picks a game seed (32 bits) when the game is created. A joiner
receives it in `GameJoined.Seed` and in the d2gs `LoadAct` packet (together
with `GameFlags`), so every client generates the same world.
`d2realm.LevelSeed(gameSeed, levelID)` (EXTENSION) derives the per-level seed
all clients must use. A client that changes level sends `LevelChange`; the
other members get `PlayerLevel`, and a later joiner gets one for each player
whose position is known.

### Drop-in and drop-out

Join: the joiner gets `GameJoined`, `GameFlags`, `LoadAct`, then a `PlayerInGame`
(0x59) for every present player (and `PlayerLevel` where known); present
players get the joiner's `PlayerInGame` and a system line. Leave (0x69) or a
disconnect: the others get `PlayerLeave` (0x5c) and a system line. After 0x69
the player gets `Result{op 0x69, ok}` and is back in the lobby. Unit ids are
server-wide counters, unique while connected.

## Characters

The server never trusts a client's character. `UploadChar` runs the
`d2s` parser (checksum, magic, version, size, body and, with `-tables`, the
item list) and these range checks (open-realm style; the bounds are
conservative upper limits and unverified against the original):

- header: valid name, class, level 1 to 99, expansion classes only with the
  expansion flag, a new (body-less) character must be level 1;
- stats: the stat level equals the header level; strength, energy, dexterity,
  vitality below 1024; their sum plus unspent points at most 90 + 5 per level
  above 1; skill points spent plus unspent at most (level - 1) + 12; gold at
  most 10000 per level; stashed gold at most 2.5 million; experience below
  2^32;
- items (with tables): known item code, item level at most 99, quality 1 to 8,
  at most 6 sockets, quantity at most 511, at most 2000 items.

A valid file is saved to `<saves>/<account>/<character>.d2s` (names lower
case, atomic write) and becomes the connection's active character. A
character name belongs to the first account that saved it. Re-uploading a
stored character may not change its class or hardcore/expansion flags. While
in a game only the game's own character may be re-uploaded (a save point).
`ListChars` returns the account's stored characters, `SelectChar` activates one
and returns its file in `CharData`, so a client can play without a local save.

The tests in `d2networking/d2realm` use the real level-94 save and tables
when `D2S_SAMPLE_BODY` and `D2_TABLES` are set, and skip otherwise.

## Gameplay (package d2networking/d2mp)

Every game owns an authoritative simulation that the realm steps at 25 Hz in
real time. Clients send intent, the realm decides, every member receives the
events of its own level. Nothing here is verified against the original server;
the numbers in `d2mp.DefaultRules` (speeds, damage, monsters, drops) are
placeholders so that the netcode can run without game files, and an engine host
supplies real rules through `Config.Rules`.

Client to server, d2gs packets (ids and layouts verified in `d2gs`):
0x01/0x03 walk/run, 0x05/0x0c cast left/right skill on a location, 0x3c select
skill, 0x13 interact (attack a monster, use an object), 0x16 pick up, 0x17 drop.
Everything else is `Command` (0x88, EXTENSION): respawn, waypoint, party
invite/accept/leave, trade request/respond/offer/accept/cancel.

Server to client: `World` (0x98, EXTENSION) batches of events: Tick (server
ms), Spawn, Seg (a movement segment: from, to, speed, start time), Attack,
Hit, Death, Remove, Object, Level (your hero changed level), Vitals, XP, Party,
Trade, Inv, Msg. For the other heroes the realm also sends the native 0x0f
PlayerMove and 0x4d UnitSkillOnLocation packets (informational).

Movement is a segment, never a stream of positions: the server and every
replica evaluate the same `Seg.PosAt`, so they agree exactly. Remote units are
drawn `InterpMs` (100) behind a server clock estimated from the fastest recent
packets; the local hero walks at once (prediction) and a matching server
segment is ignored, a different one is blended in over 200 ms (beyond 3 tiles
it snaps). Obstacles are walked around by chaining segments (A*).

Level seeds: `d2realm.LevelSeed(gameSeed, level)` feeds `Rules.Level`, so every
client builds the same layout; the monsters and objects of a level are created
by the server when the first hero enters, from that layout.

What is synced: heroes (position, walk/run, skill casts, life, death, respawn
in town, level, party), monsters (spawn, chase, attack, hit, death, corpse
removal), missiles (spawn, flight, impact, splash), ground items (monster and
chest drops, pick-up, drop), chests, town portals, level exits, waypoints,
party experience split (`d2party.ShareKillXP`), trade (items and gold, accept
lock, cancel on move/death/leave), drop-in and drop-out.

Not synced (not modelled): equipment and stats beyond life, skills' real
effects (Rules decides), player versus player, mercenaries and summons,
quests and NPC dialogues, level generation from the game's DRLG (the engine
client keeps generating its own map from the seed; the headless layout is a
placeholder grid), per-difficulty scaling. The engine's game screen is connected through
`d2client/d2realmclient` (next section).

Tests: `go test ./d2networking/d2mp ./d2networking/d2realm` (in-process, several
clients on the loopback, no windows). `scripts/mp-realm-scenario.sh [n]` hosts
a game with `cmd/od2server` and runs n `cmd/od2mpbot` processes through a
party, an exit portal and a fight, then compares their worlds.

## The game screen on the realm

Network games started from the menu (Multiplayer, TCP/IP, Host Game / Join Game)
play on the realm unless `OD2_PROTO=d2gs` or `json` selects the older direct
connection (`OD2_PROTO=realm` or unset = realm). `OD2_HOST=1` / `OD2_JOIN=host`
with `OD2_AUTOGAME` behave the same.

- The **host** process runs a `d2realm.Server` inside itself (port `OD2_PORT`,
  default 6669, bind `OD2_BIND` or 0.0.0.0), connects to it like any client and
  creates the game `od2`. A **joiner** dials the host (retrying for
  `OD2_JOIN_RETRY` seconds) and joins `od2`. `d2realmclient.Connection` is the
  game client's `ServerConnection`; `Bridge` does the translation.
- Rules: `d2mp.EngineRules`, an open arena in the engine's own tile
  coordinates with the spawn on the engine's start tile, so heroes agree with the
  engine about positions. Town holds `OD2_REALM_DUMMIES` (default 3) passive
  training dummies. All numbers are the DefaultRules placeholders (unverified).
- Heroes: the engine keeps walking its own map. Remote heroes arrive as the
  engine's AddPlayer / MovePlayer / CastSkill / Chat / disconnect packets built
  from World events (movement is released at its server time plus the 100 ms
  interpolation delay). The hero's `.d2s` is uploaded when the in-process realm
  accepts it, else a new character of the class stands in for it.
- Monsters: `RealmUnit` packets create **mirror monsters** (`d2monsters`
  `SpawnMirror`): normal entities with no AI and no local damage. A blow of the
  local hero is sent as 0x13 Interact; Hit and Death events come back
  (`MirrorHP`, `MirrorKill`: death animation and sound, no local XP or loot).
- Console: `mpkill` (fight the nearest realm monster), `mpworld` (log
  `REALM WORLD ... digest=`; equal digests = equal worlds).
- Level changes and party (code and unit tests with two simulated clients, not yet seen in two game windows): the
  engine's ChangeLevel packet becomes the realm's LevelChange (act numbered from 0), `PlayerLevel` is kept per hero
  (`Bridge.PeerLevel`, `SameLevel`) and the party commands invite / accept / leave become `d2mp.Command`s
  (decline and hostility have no realm command and are logged). Party ids from the simulation come back to the
  engine as a `RosterUpdate` packet (`d2realmclient/social.go`). A refused join is explained in the log
  (`JOIN refused: ...`); only "game not found" is retried.
- **One world per level.** The simulation (`d2mp.Sim`) already kept units, monsters, objects, missiles and items per
  level id and only queued events for the viewers in that level. What was missing was the handoff: the engine walks
  stairs and doors on its own map, and the realm only noted it for the lobby, so the hero's simulation unit stayed in
  town. Now `LevelChange` calls `Sim.ChangeLevel` (level 1..136; the hero leaves its old level with an `EvRemove`,
  appears at the new level's spawn, receives that level's units, trades are cancelled; the same level is a no-op, an
  unknown level is refused with `bad level`). A hero therefore only receives, sees and affects units of its level:
  attacking or picking up a unit id of another level does nothing, and combat events of another level are never sent.
  Levels are generated lazily, monsters of a level with no hero stand still. Portals and waypoints in the simulation
  (`Sim.UseWaypoint`, town portal, `Command`) use the same path.
- Visibility between levels: a hero in another level is not shown. The bridge removes the engine player (a
  disconnect packet) when the hero leaves the viewer's level, and announces it again (AddPlayer) when it enters;
  when the local hero changes level the bridge drops all remote heroes and mirror monsters it had.
- Party and roster stay global: `EvParty` goes to everybody, and a new `EvHero` (identity, class, level id, party) tells
  viewers in OTHER levels about a hero when it joins or changes level (viewers in the same level get the usual
  `EvSpawn`, so with every hero in one level nothing extra is sent and the traffic is unchanged). `Replica.Heroes()` is
  the global list the bridge builds the `RosterUpdate` from (area = the hero's level id). Invite/accept/leave work
  across levels; trade and the shared kill XP still need the partner in the same level.
- Not wired: the trade window in the game screen, per-difficulty monsters, monster types beyond the placeholder list,
  a dedicated `od2server` with engine rules (it uses DefaultRules, whose spawn differs from the engine's). The older
  `d2networking/d2server` (legacy `OD2_PROTO=d2gs|json`) is unchanged and still has no per-level worlds.
- Tests: `d2mp` `TestPerLevelWorldsTwoClients`, `TestPerLevelRosterJoinAndParty`; `d2realm` `TestLevelChangeSplitsTheWorld`
  (two TCP clients); `d2realmclient` `TestBridgeHidesHeroInOtherLevel` (two bridges). All `go test -race`.
- Scenarios to run ALONE before this lands (not run yet; the machine was busy): `96-multiplayer`,
  `9j-realm-multiplayer` (the realm path; checks digests at four checkpoints, in one level), and
  `9d-party-trade` / `9f-pvp-skills-ear` for the party, trade and PvP paths.
- Scenario `scripts/verify.d/9j-realm-multiplayer.sh`: two windows, both started
  at the main menu through `OD2_AUTOFLOW` (`host`, `join:<addr>`), numeric
  checks at four checkpoints (digests, dead counts, kill units and killers).
