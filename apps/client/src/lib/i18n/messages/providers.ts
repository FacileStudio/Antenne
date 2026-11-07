import type { Row } from '../row';

/** Provider screens: list, editor and detail. */
export const providers = {
	'providers.title': [
		'Providers',
		'Sources',
		'Fuentes',
		'Quellen'
	] as Row,
	'providers.subtitle': [
		'The sources Antenne watches, polls or receives on.',
		'Les sources qu’Antenne surveille, interroge ou reçoit.',
		'Las fuentes que Antenne vigila, consulta o recibe.',
		'Die Quellen, die Antenne überwacht, abfragt oder empfängt.'
	] as Row,
	'providers.newType': [
		'New provider type',
		'Type de nouvelle source',
		'Tipo de nueva fuente',
		'Typ der neuen Quelle'
	] as Row,
	'providers.add': [
		'Add provider',
		'Ajouter une source',
		'Añadir fuente',
		'Quelle hinzufügen'
	] as Row,
	'providers.emptyTitle': [
		'No provider configured yet',
		'Aucune source configurée pour l’instant',
		'Aún no hay ninguna fuente configurada',
		'Noch keine Quelle eingerichtet'
	] as Row,
	'providers.emptyText': [
		'A provider is a source Antenne watches or listens to: a website, an inbox, a feed or an incoming webhook.',
		'Une source est ce qu’Antenne surveille ou écoute : un site web, une boîte mail, un flux ou un webhook entrant.',
		'Una fuente es lo que Antenne vigila o escucha: un sitio web, un buzón, un feed o un webhook entrante.',
		'Eine Quelle ist etwas, das Antenne überwacht oder abhört: eine Website, ein Postfach, ein Feed oder ein eingehender Webhook.'
	] as Row,
	'providers.noMatchTitle': [
		'No provider matches the current tag filter',
		'Aucune source ne correspond au filtre de tags',
		'Ninguna fuente coincide con el filtro de etiquetas',
		'Keine Quelle passt zum Tag-Filter'
	] as Row,
	'providers.noMatchText': [
		'Every selected tag has to be on the provider for it to show up here.',
		'Chaque tag sélectionné doit figurer sur la source pour qu’elle apparaisse ici.',
		'Cada etiqueta seleccionada debe estar en la fuente para que aparezca aquí.',
		'Jeder gewählte Tag muss an der Quelle hängen, damit sie hier erscheint.'
	] as Row,
	'providers.clearFilter': [
		'Clear the filter',
		'Effacer le filtre',
		'Borrar el filtro',
		'Filter zurücksetzen'
	] as Row,
	'providers.editTitle': [
		'Edit {name}',
		'Modifier {name}',
		'Editar {name}',
		'{name} bearbeiten'
	] as Row,
	'providers.editFallback': [
		'Edit provider',
		'Modifier la source',
		'Editar fuente',
		'Quelle bearbeiten'
	] as Row,
	'providers.testConnection': [
		'Test connection',
		'Tester la connexion',
		'Probar la conexión',
		'Verbindung testen'
	] as Row,
	'providers.connectionOk': [
		'Connection OK for {name}.',
		'Connexion OK pour {name}.',
		'Conexión correcta para {name}.',
		'Verbindung zu {name} erfolgreich.'
	] as Row,
	'providers.connectionFailed': [
		'The connection test failed',
		'Le test de connexion a échoué',
		'La prueba de conexión falló',
		'Der Verbindungstest ist fehlgeschlagen'
	] as Row,
	'providers.deleteTitle': [
		'Delete this provider?',
		'Supprimer cette source ?',
		'¿Eliminar esta fuente?',
		'Diese Quelle löschen?'
	] as Row,
	'providers.deleteText': [
		'{name} stops being watched and every delivery target routing on it, by id or by tag, loses that source. Events already in the log are kept.',
		'{name} n’est plus surveillée et chaque cible de livraison qui la route, par identifiant ou par tag, perd cette source. Les événements déjà dans le journal sont conservés.',
		'{name} deja de vigilarse y cada destino de entrega que la use, por id o por etiqueta, pierde esa fuente. Los eventos que ya están en el registro se conservan.',
		'{name} wird nicht mehr überwacht, und jedes Zustellziel, das darauf routet, per ID oder Tag, verliert diese Quelle. Ereignisse im Protokoll bleiben erhalten.'
	] as Row,
	'providers.back': [
		'Back to providers',
		'Retour aux sources',
		'Volver a las fuentes',
		'Zurück zu den Quellen'
	] as Row,
	'providers.editProvider': [
		'Edit provider',
		'Modifier la source',
		'Editar fuente',
		'Quelle bearbeiten'
	] as Row,
	'providers.events': [
		'Events',
		'Événements',
		'Eventos',
		'Ereignisse'
	] as Row,
	'providers.delivered': [
		'Delivered',
		'Livrés',
		'Entregados',
		'Zugestellt'
	] as Row,
	'providers.failed': [
		'Failed',
		'Échecs',
		'Fallidos',
		'Fehlgeschlagen'
	] as Row,
	'providers.eventLog': [
		'Event log',
		'Journal des événements',
		'Registro de eventos',
		'Ereignisprotokoll'
	] as Row,
	'providers.eventLogText': [
		'The last 100 events this provider produced.',
		'Les 100 derniers événements produits par cette source.',
		'Los últimos 100 eventos producidos por esta fuente.',
		'Die letzten 100 Ereignisse dieser Quelle.'
	] as Row,
	'providers.refresh': [
		'Refresh the event log',
		'Actualiser le journal',
		'Actualizar el registro',
		'Protokoll aktualisieren'
	] as Row,
	'providers.loadFailed': [
		'Failed to load the event log',
		'Impossible de charger le journal',
		'No se pudo cargar el registro',
		'Das Protokoll konnte nicht geladen werden'
	] as Row,
	'providers.saveFailed': [
		'Failed to save changes',
		'Impossible d’enregistrer les modifications',
		'No se pudieron guardar los cambios',
		'Änderungen konnten nicht gespeichert werden'
	] as Row,
	'providers.goneTitle': [
		'This provider no longer exists',
		'Cette source n’existe plus',
		'Esta fuente ya no existe',
		'Diese Quelle gibt es nicht mehr'
	] as Row,
	'providers.goneText': [
		'It was deleted, or the link points at an id this Antenne has never known.',
		'Elle a été supprimée, ou le lien pointe vers un identifiant que cette instance ne connaît pas.',
		'Se eliminó, o el enlace apunta a un id que esta instancia no conoce.',
		'Sie wurde gelöscht, oder der Link verweist auf eine ID, die diese Instanz nicht kennt.'
	] as Row,
	'provider.enabledHelp': [
		'A disabled provider stops polling and stops answering on its path.',
		'Une source désactivée arrête d’interroger et de répondre sur son chemin.',
		'Una fuente desactivada deja de consultar y de responder en su ruta.',
		'Eine deaktivierte Quelle fragt nicht mehr ab und antwortet nicht mehr auf ihrem Pfad.'
	] as Row,
	'provider.name': [
		'Name',
		'Nom',
		'Nombre',
		'Name'
	] as Row,
	'provider.tags': [
		'Tags',
		'Tags',
		'Etiquetas',
		'Tags'
	] as Row,
	'provider.tagsHelp': [
		'A delivery target can subscribe by tag instead of naming this provider.',
		'Une cible de livraison peut s’abonner par tag plutôt que de nommer cette source.',
		'Un destino de entrega puede suscribirse por etiqueta en lugar de nombrar esta fuente.',
		'Ein Zustellziel kann per Tag abonnieren, statt diese Quelle zu nennen.'
	] as Row,
	'provider.url': [
		'URL',
		'URL',
		'URL',
		'URL'
	] as Row,
	'provider.checkInterval': [
		'Check interval',
		'Intervalle de vérification',
		'Intervalo de comprobación',
		'Prüfintervall'
	] as Row,
	'provider.checkIntervalHelp': [
		'In seconds. Two consecutive failures raise an alert.',
		'En secondes. Deux échecs consécutifs déclenchent une alerte.',
		'En segundos. Dos fallos consecutivos generan una alerta.',
		'In Sekunden. Zwei Fehlschläge in Folge lösen einen Alarm aus.'
	] as Row,
	'provider.feedUrl': [
		'Feed URL',
		'URL du flux',
		'URL del feed',
		'Feed-URL'
	] as Row,
	'provider.pollInterval': [
		'Poll interval',
		'Intervalle d’interrogation',
		'Intervalo de consulta',
		'Abfrageintervall'
	] as Row,
	'provider.pollIntervalRss': [
		'In seconds. The first poll only records what is already published, so a restart does not replay the feed.',
		'En secondes. La première interrogation enregistre seulement ce qui est déjà publié, un redémarrage ne rejoue donc pas le flux.',
		'En segundos. La primera consulta solo registra lo ya publicado, así que un reinicio no repite el feed.',
		'In Sekunden. Die erste Abfrage erfasst nur bereits Veröffentlichtes, ein Neustart spielt den Feed also nicht erneut ab.'
	] as Row,
	'provider.pollIntervalImap': [
		'In seconds.',
		'En secondes.',
		'En segundos.',
		'In Sekunden.'
	] as Row,
	'provider.fullContent': [
		'Include the full content',
		'Inclure le contenu complet',
		'Incluir el contenido completo',
		'Vollständigen Inhalt einbeziehen'
	] as Row,
	'provider.host': [
		'Host',
		'Hôte',
		'Host',
		'Host'
	] as Row,
	'provider.port': [
		'Port',
		'Port',
		'Puerto',
		'Port'
	] as Row,
	'provider.implicitTls': [
		'Implicit TLS',
		'TLS implicite',
		'TLS implícito',
		'Implizites TLS'
	] as Row,
	'provider.implicitTlsHelp': [
		'Off still upgrades with STARTTLS when the server offers it.',
		'Désactivé, la connexion passe quand même en STARTTLS si le serveur le propose.',
		'Desactivado, la conexión sigue pasando a STARTTLS si el servidor lo ofrece.',
		'Ausgeschaltet wird trotzdem auf STARTTLS umgestellt, wenn der Server es anbietet.'
	] as Row,
	'provider.username': [
		'Username',
		'Nom d’utilisateur',
		'Usuario',
		'Benutzername'
	] as Row,
	'provider.password': [
		'Password',
		'Mot de passe',
		'Contraseña',
		'Passwort'
	] as Row,
	'provider.emailBody': [
		'Include the email body',
		'Inclure le corps du mail',
		'Incluir el cuerpo del correo',
		'E-Mail-Text einbeziehen'
	] as Row,
	'provider.path': [
		'Path',
		'Chemin',
		'Ruta',
		'Pfad'
	] as Row,
	'provider.pathHelp': [
		'The URL this provider receives on. It is registered in the sending system, so changing it means changing it there too.',
		'L’URL sur laquelle cette source reçoit. Elle est enregistrée dans le système émetteur : la changer impose de la changer là-bas aussi.',
		'La URL en la que esta fuente recibe. Está registrada en el sistema emisor, así que cambiarla implica cambiarla también allí.',
		'Die URL, unter der diese Quelle empfängt. Sie ist im sendenden System hinterlegt, eine Änderung muss dort ebenfalls erfolgen.'
	] as Row,
	'provider.secretHeader': [
		'Secret header',
		'En-tête secret',
		'Cabecera secreta',
		'Geheim-Header'
	] as Row,
	'provider.secretValue': [
		'Secret value',
		'Valeur secrète',
		'Valor secreto',
		'Geheimer Wert'
	] as Row,
	'provider.secretValueHelp': [
		'Empty accepts unauthenticated calls on this path.',
		'Vide, les appels non authentifiés sont acceptés sur ce chemin.',
		'Vacío, se aceptan llamadas sin autenticar en esta ruta.',
		'Leer werden unauthentifizierte Aufrufe auf diesem Pfad akzeptiert.'
	] as Row,
	'provider.typeWebsite': [
		'Website',
		'Site web',
		'Sitio web',
		'Website'
	] as Row,
	'provider.typeGeneric': [
		'Generic',
		'Générique',
		'Genérico',
		'Allgemein'
	] as Row,
	'tags.placeholder': [
		'Type a tag and press Enter',
		'Saisissez un tag et appuyez sur Entrée',
		'Escribe una etiqueta y pulsa Intro',
		'Tag eingeben und Enter drücken'
	] as Row,
	'tags.remove': [
		'Remove the {tag} tag',
		'Retirer le tag {tag}',
		'Quitar la etiqueta {tag}',
		'Tag {tag} entfernen'
	] as Row,
	'list.add': [
		'Add',
		'Ajouter',
		'Añadir',
		'Hinzufügen'
	] as Row,
	'list.remove': [
		'Remove this entry',
		'Retirer cette entrée',
		'Quitar esta entrada',
		'Diesen Eintrag entfernen'
	] as Row
};
