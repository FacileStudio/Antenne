# Deploy Nook Sur La Gare

## Variables d'environnement

Configure ces variables dans Dockploy:

- `MATRIX_HOMESERVER=https://matrix.org`
- `MATRIX_ACCESS_TOKEN=<ton token Matrix>`
- `MATRIX_ROOM_ID=!pNpRDBUtbEKxTxmpEf:matrix.org`
- `CONFIG_PATH=/data/config.json`
- `NOOK_STORAGE_PATH=/data/nook-storage.json`

## Volume

Monte un volume persistant sur `/data`.

Ce volume contient:

- `/data/config.json`
- `/data/nook-storage.json`

## Contenu de `/data/config.json`

```json
{
  "sites": [
    {
      "name": "Facile Studio",
      "url": "https://facile.studio",
      "interval": 60
    },
    {
      "name": "Laura Herve",
      "url": "https://lauraherve.com",
      "interval": 60
    },
    {
      "name": "Marcel",
      "url": "https://marcel.my",
      "interval": 60
    }
  ],
  "timeout": 10
}
```

## Build

Dockerfile déjà prêt. Dockploy peut build le repo directement.

## Run check

Au démarrage, Nook doit logguer:

```text
Nook started — monitoring 3 site(s)
```

Puis une ligne par site.

## Notes utiles

- Les réponses HTTP `< 400` sont considérées comme `up`, donc les redirects ne déclenchent pas de faux down.
- Le bot n'a pas besoin de sync Matrix complet pour envoyer des alertes.
- Si Nook tourne sur la même machine que les sites monitorés, il ne verra pas une panne totale de la machine elle-même.
