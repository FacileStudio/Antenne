import type { Row } from '../row';

/** Settings and spaces screens. */
export const settings = {
	'settings.title': [
		'Settings',
		'Réglages',
		'Ajustes',
		'Einstellungen'
	] as Row,
	'settings.subtitle': [
		'This instance’s configuration. Changes save automatically.',
		'La configuration de cette instance. Les modifications sont enregistrées automatiquement.',
		'La configuración de esta instancia. Los cambios se guardan automáticamente.',
		'Die Konfiguration dieser Instanz. Änderungen werden automatisch gespeichert.'
	] as Row,
	'settings.sections': [
		'Settings sections',
		'Sections des réglages',
		'Secciones de ajustes',
		'Einstellungsbereiche'
	] as Row,
	'settings.profile': [
		'Profile',
		'Profil',
		'Perfil',
		'Profil'
	] as Row,
	'settings.appearance': [
		'Appearance',
		'Apparence',
		'Apariencia',
		'Darstellung'
	] as Row,
	'settings.bus': [
		'Bus',
		'Bus',
		'Bus',
		'Bus'
	] as Row,
	'settings.events': [
		'Events',
		'Événements',
		'Eventos',
		'Ereignisse'
	] as Row,
	'settings.advanced': [
		'Advanced',
		'Avancé',
		'Avanzado',
		'Erweitert'
	] as Row,
	'profile.account': [
		'Account',
		'Compte',
		'Cuenta',
		'Konto'
	] as Row,
	'profile.accountText': [
		'The identity this browser is signed in with.',
		'L’identité avec laquelle ce navigateur est connecté.',
		'La identidad con la que este navegador ha iniciado sesión.',
		'Die Identität, mit der dieser Browser angemeldet ist.'
	] as Row,
	'profile.signedInAs': [
		'Signed in as',
		'Connecté en tant que',
		'Sesión iniciada como',
		'Angemeldet als'
	] as Row,
	'profile.role': [
		'Role',
		'Rôle',
		'Rol',
		'Rolle'
	] as Row,
	'profile.authentication': [
		'Authentication',
		'Authentification',
		'Autenticación',
		'Authentifizierung'
	] as Row,
	'profile.authenticationText': [
		'Antenne signs people in through Registre only.',
		'Antenne ne connecte les personnes que par Registre.',
		'Antenne solo inicia sesión a través de Registre.',
		'Antenne meldet Personen ausschließlich über Registre an.'
	] as Row,
	'profile.sso': [
		'Signed in with Facile',
		'Connecté avec Facile',
		'Sesión iniciada con Facile',
		'Mit Facile angemeldet'
	] as Row,
	'profile.instance': [
		'Instance',
		'Instance',
		'Instancia',
		'Instanz'
	] as Row,
	'profile.session': [
		'Session',
		'Session',
		'Sesión',
		'Sitzung'
	] as Row,
	'profile.sessionText': [
		'This browser’s session on this instance.',
		'La session de ce navigateur sur cette instance.',
		'La sesión de este navegador en esta instancia.',
		'Die Sitzung dieses Browsers auf dieser Instanz.'
	] as Row,
	'profile.logout': [
		'Log out',
		'Se déconnecter',
		'Cerrar sesión',
		'Abmelden'
	] as Row,
	'profile.logoutText': [
		'Ends the session and returns to the login screen. Unsaved settings drafts are discarded.',
		'Termine la session et revient à l’écran de connexion. Les brouillons non enregistrés sont perdus.',
		'Termina la sesión y vuelve a la pantalla de inicio de sesión. Se descartan los borradores sin guardar.',
		'Beendet die Sitzung und kehrt zur Anmeldung zurück. Nicht gespeicherte Entwürfe gehen verloren.'
	] as Row,
	'appearance.theme': [
		'Theme',
		'Thème',
		'Tema',
		'Design'
	] as Row,
	'appearance.themeText': [
		'Applied to this browser only. Antenne stores nothing about it server-side.',
		'Appliqué à ce navigateur uniquement. Antenne ne stocke rien côté serveur.',
		'Se aplica solo a este navegador. Antenne no guarda nada en el servidor.',
		'Gilt nur für diesen Browser. Antenne speichert serverseitig nichts dazu.'
	] as Row,
	'appearance.scheme': [
		'Colour scheme',
		'Palette de couleurs',
		'Esquema de color',
		'Farbschema'
	] as Row,
	'appearance.schemeText': [
		'System follows the operating system’s preference.',
		'Système suit la préférence du système d’exploitation.',
		'Sistema sigue la preferencia del sistema operativo.',
		'System folgt der Einstellung des Betriebssystems.'
	] as Row,
	'appearance.system': [
		'System',
		'Système',
		'Sistema',
		'System'
	] as Row,
	'appearance.light': [
		'Light',
		'Clair',
		'Claro',
		'Hell'
	] as Row,
	'appearance.dark': [
		'Dark',
		'Sombre',
		'Oscuro',
		'Dunkel'
	] as Row,
	'sysevents.title': [
		'System events',
		'Événements système',
		'Eventos del sistema',
		'Systemereignisse'
	] as Row,
	'sysevents.text': [
		'The events Antenne emits about itself, on top of the ones its providers report.',
		'Les événements qu’Antenne émet sur elle-même, en plus de ceux que ses sources rapportent.',
		'Los eventos que Antenne emite sobre sí misma, además de los que informan sus fuentes.',
		'Die Ereignisse, die Antenne über sich selbst ausgibt, zusätzlich zu denen ihrer Quellen.'
	] as Row,
	'sysevents.sendTest': [
		'Send test alert',
		'Envoyer une alerte de test',
		'Enviar alerta de prueba',
		'Testalarm senden'
	] as Row,
	'sysevents.startup': [
		'Antenne started',
		'Antenne a démarré',
		'Antenne se inició',
		'Antenne wurde gestartet'
	] as Row,
	'sysevents.providerEnabled': [
		'A provider was enabled',
		'Une source a été activée',
		'Se activó una fuente',
		'Eine Quelle wurde aktiviert'
	] as Row,
	'sysevents.providerDisabled': [
		'A provider was disabled',
		'Une source a été désactivée',
		'Se desactivó una fuente',
		'Eine Quelle wurde deaktiviert'
	] as Row,
	'sysevents.providerAdded': [
		'A provider was added',
		'Une source a été ajoutée',
		'Se añadió una fuente',
		'Eine Quelle wurde hinzugefügt'
	] as Row,
	'sysevents.providerRemoved': [
		'A provider was removed',
		'Une source a été retirée',
		'Se eliminó una fuente',
		'Eine Quelle wurde entfernt'
	] as Row,
	'sysevents.deliveryFailure': [
		'A delivery target failed',
		'Une cible de livraison a échoué',
		'Falló un destino de entrega',
		'Ein Zustellziel ist fehlgeschlagen'
	] as Row,
	'sysevents.alertTitle': [
		'Enabling an event is not enough on its own',
		'Activer un événement ne suffit pas',
		'Activar un evento no basta por sí solo',
		'Ein Ereignis zu aktivieren reicht nicht aus'
	] as Row,
	'sysevents.alertText': [
		'Delivery routing is opt-in with no fallthrough: a system event only reaches a target that names the reserved provider id {id} in its provider filter. A target filtered on tags alone never sees one.',
		'Le routage est explicite, sans repli : un événement système n’atteint qu’une cible qui nomme l’identifiant de source réservé {id} dans son filtre de sources. Une cible filtrée uniquement par tags n’en voit jamais.',
		'El enrutamiento es explícito y sin respaldo: un evento del sistema solo llega a un destino que nombra el id de fuente reservado {id} en su filtro de fuentes. Un destino filtrado solo por etiquetas nunca lo ve.',
		'Das Routing ist ausdrücklich und ohne Auffangregel: Ein Systemereignis erreicht nur ein Ziel, das die reservierte Quellen-ID {id} in seinem Quellenfilter nennt. Ein Ziel, das nur nach Tags filtert, sieht nie eines.'
	] as Row,
	'advanced.tuning': [
		'Runtime tuning',
		'Réglages d’exécution',
		'Ajuste en ejecución',
		'Laufzeit-Feineinstellung'
	] as Row,
	'advanced.tuningText': [
		'Applied automatically on runtime reload.',
		'Appliqués automatiquement au rechargement.',
		'Se aplican automáticamente al recargar.',
		'Wird beim Neuladen automatisch angewendet.'
	] as Row,
	'advanced.timeout': [
		'Request timeout',
		'Délai d’attente des requêtes',
		'Tiempo de espera de solicitudes',
		'Anfrage-Zeitlimit'
	] as Row,
	'advanced.timeoutText': [
		'Seconds a website check waits before it counts as down.',
		'Secondes d’attente d’une vérification de site avant de la compter comme en panne.',
		'Segundos que espera una comprobación de sitio antes de darlo por caído.',
		'Sekunden, die eine Website-Prüfung wartet, bevor sie als ausgefallen gilt.'
	] as Row,
	'advanced.port': [
		'Listener port',
		'Port d’écoute',
		'Puerto de escucha',
		'Listener-Port'
	] as Row,
	'advanced.portText': [
		'Takes effect on the next restart. Every published webhook URL and every pool client points at it, so the deployment has to agree.',
		'Prend effet au prochain redémarrage. Chaque URL de webhook publiée et chaque client du pool y pointe : le déploiement doit être cohérent.',
		'Surte efecto en el próximo reinicio. Cada URL de webhook publicada y cada cliente del pool apunta a él, así que el despliegue debe coincidir.',
		'Wirkt ab dem nächsten Neustart. Jede veröffentlichte Webhook-URL und jeder Pool-Client verweist darauf, das Deployment muss also passen.'
	] as Row,
	'advanced.runtime': [
		'Runtime',
		'Exécution',
		'Ejecución',
		'Laufzeit'
	] as Row,
	'advanced.runtimeText': [
		'What this process is actually doing.',
		'Ce que fait réellement ce processus.',
		'Lo que está haciendo realmente este proceso.',
		'Was dieser Prozess tatsächlich tut.'
	] as Row,
	'advanced.listening': [
		'Listening on',
		'Écoute sur',
		'Escuchando en',
		'Lauscht auf'
	] as Row,
	'advanced.store': [
		'Settings store',
		'Stockage des réglages',
		'Almacén de ajustes',
		'Einstellungsspeicher'
	] as Row,
	'advanced.source': [
		'Source',
		'Origine',
		'Origen',
		'Herkunft'
	] as Row,
	'advanced.sourceSaved': [
		'Saved from the dashboard',
		'Enregistré depuis le tableau de bord',
		'Guardado desde el panel',
		'Im Dashboard gespeichert'
	] as Row,
	'advanced.sourceDefaults': [
		'Defaults',
		'Valeurs par défaut',
		'Valores por defecto',
		'Standardwerte'
	] as Row,
	'advanced.lastReload': [
		'Last reload',
		'Dernier rechargement',
		'Última recarga',
		'Letztes Neuladen'
	] as Row,
	'advanced.configuration': [
		'Configuration',
		'Configuration',
		'Configuración',
		'Konfiguration'
	] as Row,
	'advanced.configurationText': [
		'The whole settings document as one JSON file.',
		'Tout le document de réglages dans un seul fichier JSON.',
		'Todo el documento de ajustes en un único archivo JSON.',
		'Das gesamte Einstellungsdokument als eine JSON-Datei.'
	] as Row,
	'advanced.export': [
		'Export',
		'Export',
		'Exportar',
		'Export'
	] as Row,
	'advanced.exportText': [
		'Secrets come out redacted, so the file is safe to keep and to commit.',
		'Les secrets sont masqués : le fichier peut être conservé et versionné sans risque.',
		'Los secretos salen ocultos, así que el archivo se puede guardar y versionar sin riesgo.',
		'Geheimnisse werden geschwärzt, die Datei kann also bedenkenlos aufbewahrt und eingecheckt werden.'
	] as Row,
	'advanced.exportButton': [
		'Export config',
		'Exporter la configuration',
		'Exportar configuración',
		'Konfiguration exportieren'
	] as Row,
	'advanced.danger': [
		'Danger zone',
		'Zone dangereuse',
		'Zona de peligro',
		'Gefahrenbereich'
	] as Row,
	'advanced.dangerText': [
		'Irreversible from the dashboard. Keep an export first.',
		'Irréversible depuis le tableau de bord. Gardez d’abord un export.',
		'Irreversible desde el panel. Guarda antes una exportación.',
		'Im Dashboard nicht umkehrbar. Erstelle zuerst einen Export.'
	] as Row,
	'advanced.import': [
		'Import a configuration',
		'Importer une configuration',
		'Importar una configuración',
		'Konfiguration importieren'
	] as Row,
	'advanced.importText': [
		'Replaces every provider, delivery target and system setting, then saves immediately.',
		'Remplace chaque source, cible de livraison et réglage système, puis enregistre immédiatement.',
		'Reemplaza cada fuente, destino de entrega y ajuste del sistema, y guarda de inmediato.',
		'Ersetzt jede Quelle, jedes Zustellziel und jede Systemeinstellung und speichert sofort.'
	] as Row,
	'advanced.chooseFile': [
		'Choose a file…',
		'Choisir un fichier…',
		'Elegir un archivo…',
		'Datei auswählen…'
	] as Row,
	'advanced.importTitle': [
		'Replace this instance’s configuration?',
		'Remplacer la configuration de cette instance ?',
		'¿Reemplazar la configuración de esta instancia?',
		'Konfiguration dieser Instanz ersetzen?'
	] as Row,
	'advanced.importWarning': [
		'Importing {name} overwrites every provider, every delivery target and every system setting, and saves them straight away. Providers that are not in the file stop polling and their webhook paths start answering 404; delivery targets that are not in the file stop receiving alerts. Redacted secrets in the file are restored from what is stored, so a target the file does not name loses its credentials with it.',
		'L’import de {name} écrase chaque source, chaque cible de livraison et chaque réglage système, puis les enregistre aussitôt. Les sources absentes du fichier cessent d’être interrogées et leurs chemins de webhook répondent 404 ; les cibles absentes du fichier ne reçoivent plus d’alertes. Les secrets masqués du fichier sont restaurés depuis ce qui est stocké : une cible que le fichier ne nomme pas perd donc ses identifiants.',
		'Importar {name} sobrescribe cada fuente, cada destino de entrega y cada ajuste del sistema, y los guarda de inmediato. Las fuentes que no están en el archivo dejan de consultarse y sus rutas de webhook responden 404; los destinos que no están en el archivo dejan de recibir alertas. Los secretos ocultos del archivo se restauran desde lo almacenado, así que un destino que el archivo no nombra pierde sus credenciales.',
		'Der Import von {name} überschreibt jede Quelle, jedes Zustellziel und jede Systemeinstellung und speichert sie sofort. Quellen, die nicht in der Datei stehen, werden nicht mehr abgefragt und ihre Webhook-Pfade antworten mit 404; Zustellziele, die nicht in der Datei stehen, erhalten keine Alarme mehr. Geschwärzte Geheimnisse in der Datei werden aus dem Gespeicherten wiederhergestellt, ein Ziel, das die Datei nicht nennt, verliert also seine Zugangsdaten.'
	] as Row,
	'advanced.thisFile': [
		'this file',
		'ce fichier',
		'este archivo',
		'diese Datei'
	] as Row,
	'advanced.replace': [
		'Replace configuration',
		'Remplacer la configuration',
		'Reemplazar la configuración',
		'Konfiguration ersetzen'
	] as Row,
	'bus.registration': [
		'Registration',
		'Enregistrement',
		'Registro',
		'Registrierung'
	] as Row,
	'bus.registrationText': [
		'The shared secret a pool client presents at /pool/register.',
		'Le secret partagé qu’un client du pool présente sur /pool/register.',
		'El secreto compartido que un cliente del pool presenta en /pool/register.',
		'Das gemeinsame Geheimnis, das ein Pool-Client bei /pool/register vorlegt.'
	] as Row,
	'bus.secret': [
		'Registration secret',
		'Secret d’enregistrement',
		'Secreto de registro',
		'Registrierungsgeheimnis'
	] as Row,
	'bus.secretDashboard': [
		'Set from the dashboard',
		'Défini depuis le tableau de bord',
		'Definido desde el panel',
		'Im Dashboard festgelegt'
	] as Row,
	'bus.secretFallback': [
		'Falling back to ANTENNE_SECRET or "pool"',
		'Repli sur ANTENNE_SECRET ou « pool »',
		'Se usa ANTENNE_SECRET o «pool»',
		'Rückfall auf ANTENNE_SECRET oder „pool“'
	] as Row,
	'bus.logEvents': [
		'Log pool events to Activity',
		'Journaliser les événements du pool dans l’activité',
		'Registrar los eventos del pool en la actividad',
		'Pool-Ereignisse in der Aktivität protokollieren'
	] as Row,
	'bus.logEventsText': [
		'Connections, disconnections and relayed events appear in the activity feed.',
		'Les connexions, déconnexions et événements relayés apparaissent dans le flux d’activité.',
		'Las conexiones, desconexiones y eventos retransmitidos aparecen en el flujo de actividad.',
		'Verbindungen, Trennungen und weitergeleitete Ereignisse erscheinen im Aktivitätsstrom.'
	] as Row,
	'bus.connection': [
		'Connection',
		'Connexion',
		'Conexión',
		'Verbindung'
	] as Row,
	'bus.connectionText': [
		'What the pool reports right now.',
		'Ce que le pool indique en ce moment.',
		'Lo que informa el pool ahora mismo.',
		'Was der Pool gerade meldet.'
	] as Row,
	'bus.state': [
		'State',
		'État',
		'Estado',
		'Zustand'
	] as Row,
	'bus.epoch': [
		'Epoch',
		'Époque',
		'Época',
		'Epoche'
	] as Row,
	'bus.epochText': [
		'Restarts mint a new one; clients resync when it changes.',
		'Chaque redémarrage en crée une nouvelle ; les clients se resynchronisent quand elle change.',
		'Cada reinicio crea una nueva; los clientes se resincronizan cuando cambia.',
		'Jeder Neustart erzeugt eine neue; Clients synchronisieren sich neu, wenn sie sich ändert.'
	] as Row,
	'bus.liveConnections': [
		'Live connections',
		'Connexions actives',
		'Conexiones activas',
		'Aktive Verbindungen'
	] as Row,
	'bus.lastPoll': [
		'Last successful poll',
		'Dernière interrogation réussie',
		'Última consulta correcta',
		'Letzte erfolgreiche Abfrage'
	] as Row,
	'bus.lastError': [
		'Last error',
		'Dernière erreur',
		'Último error',
		'Letzter Fehler'
	] as Row,
	'bus.statsFailed': [
		'The stats endpoint did not answer.',
		'Le point d’accès des statistiques n’a pas répondu.',
		'El punto de acceso de estadísticas no respondió.',
		'Der Statistik-Endpunkt hat nicht geantwortet.'
	] as Row,
	'bus.unreachable': [
		'Unreachable, gave up after {count} attempts',
		'Injoignable, abandon après {count} tentatives',
		'Inaccesible, se abandonó tras {count} intentos',
		'Nicht erreichbar, nach {count} Versuchen aufgegeben'
	] as Row,
	'bus.reconnecting': [
		'Reconnecting, attempt {count} of {max}',
		'Reconnexion, tentative {count} sur {max}',
		'Reconectando, intento {count} de {max}',
		'Verbindung wird wiederhergestellt, Versuch {count} von {max}'
	] as Row,
	'bus.connecting': [
		'Connecting…',
		'Connexion…',
		'Conectando…',
		'Verbindung wird aufgebaut…'
	] as Row,
	'bus.listening': [
		'Listening, no app connected',
		'À l’écoute, aucune app connectée',
		'Escuchando, ninguna app conectada',
		'Lauscht, keine App verbunden'
	] as Row,
	'bus.connected': [
		{ one: 'Connected, {count} live socket', other: 'Connected, {count} live sockets' },
		{ one: 'Connecté, {count} socket actif', other: 'Connecté, {count} sockets actifs' },
		{ one: 'Conectado, {count} socket activo', other: 'Conectado, {count} sockets activos' },
		{ one: 'Verbunden, {count} aktiver Socket', other: 'Verbunden, {count} aktive Sockets' }
	] as Row,
	'bus.apps': [
		'Connected apps',
		'Apps connectées',
		'Apps conectadas',
		'Verbundene Apps'
	] as Row,
	'bus.appsText': [
		'Suite apps holding a socket, with the channels each subscribes to. An app that only produces events shows as emits only.',
		'Les apps de la suite qui tiennent un socket, avec les canaux auxquels chacune s’abonne. Une app qui ne fait que produire des événements apparaît comme « émet seulement ».',
		'Apps de la suite con un socket abierto y los canales a los que se suscribe cada una. Una app que solo produce eventos aparece como «solo emite».',
		'Suite-Apps mit offenem Socket und die Kanäle, die jede abonniert. Eine App, die nur Ereignisse erzeugt, erscheint als „sendet nur“.'
	] as Row,
	'bus.noApp': [
		'No app is connected',
		'Aucune app connectée',
		'Ninguna app conectada',
		'Keine App verbunden'
	] as Row,
	'bus.noAppText': [
		'An app appears here once it registers with the secret above and opens /pool/ws.',
		'Une app apparaît ici dès qu’elle s’enregistre avec le secret ci-dessus et ouvre /pool/ws.',
		'Una app aparece aquí cuando se registra con el secreto de arriba y abre /pool/ws.',
		'Eine App erscheint hier, sobald sie sich mit dem Geheimnis oben registriert und /pool/ws öffnet.'
	] as Row,
	'bus.emitsOnly': [
		'emits only',
		'émet seulement',
		'solo emite',
		'sendet nur'
	] as Row,
	'bus.noChannel': [
		'No channel',
		'Aucun canal',
		'Ningún canal',
		'Kein Kanal'
	] as Row,
	'spaces.title': [
		'Spaces',
		'Espaces',
		'Espacios',
		'Bereiche'
	] as Row,
	'spaces.text': [
		'Spaces and their members come from Registre, the same in every Facile app. Antenne’s providers and delivery targets stay instance-wide.',
		'Les espaces et leurs membres viennent de Registre, les mêmes dans chaque app Facile. Les sources et cibles de livraison d’Antenne restent communes à l’instance.',
		'Los espacios y sus miembros vienen de Registre, los mismos en cada app de Facile. Las fuentes y destinos de entrega de Antenne siguen siendo comunes a toda la instancia.',
		'Bereiche und ihre Mitglieder kommen aus Registre, in jeder Facile-App dieselben. Die Quellen und Zustellziele von Antenne gelten weiterhin für die ganze Instanz.'
	] as Row,
	'spaces.manage': [
		'Manage in Registre',
		'Gérer dans Registre',
		'Gestionar en Registre',
		'In Registre verwalten'
	] as Row,
	'spaces.emptyTitle': [
		'No space yet',
		'Aucun espace pour l’instant',
		'Aún no hay espacios',
		'Noch kein Bereich'
	] as Row,
	'spaces.emptyText': [
		'Create a space or join one in Registre and it shows up here.',
		'Créez ou rejoignez un espace dans Registre et il apparaît ici.',
		'Crea o únete a un espacio en Registre y aparecerá aquí.',
		'Erstelle oder betrete einen Bereich in Registre und er erscheint hier.'
	] as Row,
	'spaces.created': [
		'Created {date}',
		'Créé {date}',
		'Creado {date}',
		'Erstellt {date}'
	] as Row,
	'spaces.back': [
		'Back to spaces',
		'Retour aux espaces',
		'Volver a los espacios',
		'Zurück zu den Bereichen'
	] as Row,
	'spaces.goneTitle': [
		'This space no longer exists',
		'Cet espace n’existe plus',
		'Este espacio ya no existe',
		'Diesen Bereich gibt es nicht mehr'
	] as Row,
	'spaces.goneText': [
		'It was removed, or the id in the URL never matched one.',
		'Il a été supprimé, ou l’identifiant de l’URL n’a jamais correspondu à un espace.',
		'Se eliminó, o el id de la URL nunca coincidió con uno.',
		'Er wurde entfernt, oder die ID in der URL hat nie zu einem gepasst.'
	] as Row,
	'spaces.loadFailed': [
		'Failed to load the space',
		'Impossible de charger l’espace',
		'No se pudo cargar el espacio',
		'Der Bereich konnte nicht geladen werden'
	] as Row,
	'spaces.memberCount': [
		{ one: '{count} member', other: '{count} members' },
		{ one: '{count} membre', other: '{count} membres' },
		{ one: '{count} miembro', other: '{count} miembros' },
		{ one: '{count} Mitglied', other: '{count} Mitglieder' }
	] as Row,
	'spaces.summary': [
		'{members} · created {date}',
		'{members} · créé {date}',
		'{members} · creado {date}',
		'{members} · erstellt {date}'
	] as Row,
	'spaces.members': [
		'Members',
		'Membres',
		'Miembros',
		'Mitglieder'
	] as Row,
	'spaces.membersText': [
		'Who belongs to this space, and at what level. Invites and roles are managed in Registre.',
		'Qui appartient à cet espace, et à quel niveau. Les invitations et les rôles se gèrent dans Registre.',
		'Quién pertenece a este espacio y con qué nivel. Las invitaciones y los roles se gestionan en Registre.',
		'Wer zu diesem Bereich gehört und mit welcher Stufe. Einladungen und Rollen werden in Registre verwaltet.'
	] as Row,
	'spaces.noMember': [
		'No member yet',
		'Aucun membre pour l’instant',
		'Aún no hay miembros',
		'Noch kein Mitglied'
	] as Row,
	'spaces.noMemberText': [
		'Invite someone in Registre and they show up here.',
		'Invitez quelqu’un dans Registre et la personne apparaît ici.',
		'Invita a alguien en Registre y aparecerá aquí.',
		'Lade jemanden in Registre ein und die Person erscheint hier.'
	] as Row,
	'spaces.joined': [
		'Joined {date}',
		'Rejoint {date}',
		'Se unió {date}',
		'Beigetreten {date}'
	] as Row,
	'role.owner': [
		'owner',
		'propriétaire',
		'propietario',
		'Inhaber'
	] as Row,
	'role.admin': [
		'admin',
		'admin',
		'administrador',
		'Admin'
	] as Row,
	'role.member': [
		'member',
		'membre',
		'miembro',
		'Mitglied'
	] as Row
};
