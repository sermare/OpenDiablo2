# Act 1 playthrough log

Branch `feat/act1-playthrough`. The first hour of Act 1 is played by
`OD2_AUTOSCRIPT` (scenario `scripts/verify.d/9b-act1-playthrough.sh`, reload check
`9c-act1-reload.sh`): Rogue Encampment, Akara (Den of Evil), the gate to Blood Moor,
fights, the Den of Evil, back to Akara for the reward, save, reload from the exported
`.d2s`. Every seam found by playing is listed here with its fix.

Script steps added for this (`d2game/d2autoscript`): `walkto:exit=<level>`,
`walkto:object=<name>`, `kill:all[,<s>]`, `kill:near=<tiles>[,<s>]`,
`until:<log substring>,<timeout>`, `menu:<row>`.

Run it by hand (see `docs/macos-quickstart.md`):

```sh
OD2_REALMAPS=1 OD2_AUTOGAME=hero.d2s OD2_AUTOEXIT=1 \
  OD2_AUTOSCRIPT='wait:1;walkto:exit=2;expect:level=2;walkto:exit=8;kill:all,200;exit' ./od2
```

## Bugs found and fixed

| # | Seam | Fix | Commit |
|---|------|-----|--------|
