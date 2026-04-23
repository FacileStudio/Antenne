# Deploy Nook Sur La Gare

## Variables d'environnement

Configure ces variables dans Dockploy:

- `MATRIX_HOMESERVER=https://matrix.org`
- `MATRIX_ACCESS_TOKEN=<ton token Matrix>`
- `MATRIX_ROOM_ID=!pNpRDBUtbEKxTxmpEf:matrix.org`
- `CONFIG_PATH=/data/config.yml`

## Volume

Monte un volume persistant sur `/data`.

Ce volume contient:

- `/data/config.yml`

## Contenu de `/data/config.yml`

```yaml
timeout: 10
every: 60

sites:
  - https://facile.studio
  - https://portail.facile.studio
  - https://gare.facile.studio
  - https://abeille.facile.studio
  - https://miel.facile.studio
  - https://automate.facile.studio
  - https://sign.facile.studio
  - https://nuage.facile.studio
  - https://stats.facile.studio
  - https://coffre.facile.studio
  - https://lauraherve.com
  - https://api.lauraherve.com
  - https://backoffice.lauraherve.com
  - https://marcel.my
```

## Build

Dockerfile déjà prêt. Dockploy peut build le repo directement.
L'image utilise `oven/bun:1.3.11` et Nook parle directement à l'API Matrix par HTTP. Pas de SDK crypto natif à télécharger, donc build moins casse-gueule.

## Run check

Au démarrage, Nook doit logguer:

```text
[startup] Nook booting
[startup] Matrix startup message sent
```

Puis:

- une ligne `[monitor]` par site
- une ligne `[health]` à chaque check
- un message `Nook started` dans le salon Matrix

## Notes utiles

- Les réponses HTTP `< 400` sont considérées comme `up`, donc les redirects ne déclenchent pas de faux down.
- Le bot n'a pas besoin de sync Matrix complet pour envoyer des alertes.
- Nook est maintenant prévu pour un salon Matrix non chiffré. Il envoie directement `m.room.message` via l'API Matrix.
- Le YAML est le format canonique. L'ancien JSON marche encore en fallback pendant la transition, mais il ne mérite plus la vedette.
- Les noms affichés sont dérivés des URLs; le YAML canonique n'a besoin que d'un tableau `sites`.
- Si Nook tourne sur la même machine que les sites monitorés, il ne verra pas une panne totale de la machine elle-même.
