import type { Row } from '../row';

/** Delivery screens: targets, health and the editor. */
export const delivery = {
	'delivery.title': [
		'Delivery',
		'Livraison',
		'Entrega',
		'Zustellung'
	] as Row,
	'delivery.subtitle': [
		'Where Antenne sends what it sees. A target receives an event only if it names the provider or shares one of its tags.',
		'Où Antenne envoie ce qu’elle voit. Une cible ne reçoit un événement que si elle nomme la source ou partage l’un de ses tags.',
		'Adónde envía Antenne lo que ve. Un destino solo recibe un evento si nombra la fuente o comparte una de sus etiquetas.',
		'Wohin Antenne sendet, was sie sieht. Ein Ziel erhält ein Ereignis nur, wenn es die Quelle nennt oder einen ihrer Tags teilt.'
	] as Row,
	'delivery.newType': [
		'New target type',
		'Type de nouvelle cible',
		'Tipo de nuevo destino',
		'Typ des neuen Ziels'
	] as Row,
	'delivery.add': [
		'Add target',
		'Ajouter une cible',
		'Añadir destino',
		'Ziel hinzufügen'
	] as Row,
	'delivery.emptyTitle': [
		'No delivery target yet',
		'Aucune cible de livraison pour l’instant',
		'Aún no hay ningún destino de entrega',
		'Noch kein Zustellziel'
	] as Row,
	'delivery.emptyText': [
		'Antenne is watching, but nothing it sees leaves the box until a target exists.',
		'Antenne surveille, mais rien de ce qu’elle voit ne sort tant qu’aucune cible n’existe.',
		'Antenne vigila, pero nada de lo que ve sale hasta que exista un destino.',
		'Antenne überwacht, aber nichts verlässt das System, bis ein Ziel existiert.'
	] as Row,
	'delivery.deleteTitle': [
		'Delete this delivery target?',
		'Supprimer cette cible de livraison ?',
		'¿Eliminar este destino de entrega?',
		'Dieses Zustellziel löschen?'
	] as Row,
	'delivery.deleteText': [
		'{name} stops receiving events the moment this is saved, and any provider routed only here goes nowhere. Deleting cannot be undone.',
		'{name} cesse de recevoir des événements dès l’enregistrement, et toute source routée uniquement ici ne va nulle part. La suppression est définitive.',
		'{name} deja de recibir eventos en cuanto se guarda, y cualquier fuente enrutada solo aquí no va a ninguna parte. Eliminar no se puede deshacer.',
		'{name} erhält ab dem Speichern keine Ereignisse mehr, und jede nur hierhin geroutete Quelle läuft ins Leere. Das Löschen kann nicht rückgängig gemacht werden.'
	] as Row,
	'delivery.answered': [
		'{name} answered.',
		'{name} a répondu.',
		'{name} respondió.',
		'{name} hat geantwortet.'
	] as Row,
	'delivery.declined': [
		'{name} declined: {detail}',
		'{name} a refusé : {detail}',
		'{name} rechazó: {detail}',
		'{name} hat abgelehnt: {detail}'
	] as Row,
	'delivery.noReason': [
		'no reason given',
		'aucune raison donnée',
		'sin motivo',
		'kein Grund angegeben'
	] as Row,
	'delivery.tooMany': [
		'Too many checks. This is Antenne’s own limit, not a refusal from {name}. Retry in {seconds}s.',
		'Trop de vérifications. C’est la limite d’Antenne, pas un refus de {name}. Réessayez dans {seconds} s.',
		'Demasiadas comprobaciones. Es el límite de Antenne, no un rechazo de {name}. Reintenta en {seconds} s.',
		'Zu viele Prüfungen. Das ist das Limit von Antenne, keine Ablehnung von {name}. Erneut versuchen in {seconds} s.'
	] as Row,
	'delivery.checkFailed': [
		'The check could not run',
		'La vérification n’a pas pu s’exécuter',
		'No se pudo ejecutar la comprobación',
		'Die Prüfung konnte nicht ausgeführt werden'
	] as Row,
	'delivery.silent': [
		'No source attached',
		'Aucune source rattachée',
		'Ninguna fuente asociada',
		'Keine Quelle zugeordnet'
	] as Row,
	'delivery.silentHelp': [
		'Attach a provider or a tag, or this target receives nothing.',
		'Rattachez une source ou un tag, sinon cette cible ne reçoit rien.',
		'Asocia una fuente o una etiqueta, o este destino no recibirá nada.',
		'Ordne eine Quelle oder einen Tag zu, sonst erhält dieses Ziel nichts.'
	] as Row,
	'delivery.colTarget': [
		'Target',
		'Cible',
		'Destino',
		'Ziel'
	] as Row,
	'delivery.colType': [
		'Type',
		'Type',
		'Tipo',
		'Typ'
	] as Row,
	'delivery.colRouting': [
		'Routing',
		'Routage',
		'Enrutamiento',
		'Routing'
	] as Row,
	'delivery.colHealth': [
		'Health',
		'État',
		'Estado',
		'Zustand'
	] as Row,
	'delivery.colActions': [
		'Actions',
		'Actions',
		'Acciones',
		'Aktionen'
	] as Row,
	'delivery.activity': [
		'Activity',
		'Activité',
		'Actividad',
		'Aktivität'
	] as Row,
	'delivery.routingProviders': [
		{ one: '{count} provider', other: '{count} providers' },
		{ one: '{count} source', other: '{count} sources' },
		{ one: '{count} fuente', other: '{count} fuentes' },
		{ one: '{count} Quelle', other: '{count} Quellen' }
	] as Row,
	'delivery.routingTags': [
		{ one: '{count} tag', other: '{count} tags' },
		{ one: '{count} tag', other: '{count} tags' },
		{ one: '{count} etiqueta', other: '{count} etiquetas' },
		{ one: '{count} Tag', other: '{count} Tags' }
	] as Row,
	'delivery.editTitle': [
		'Edit {name}',
		'Modifier {name}',
		'Editar {name}',
		'{name} bearbeiten'
	] as Row,
	'delivery.editFallback': [
		'Edit target',
		'Modifier la cible',
		'Editar destino',
		'Ziel bearbeiten'
	] as Row,
	'delivery.sendTest': [
		'Send a test',
		'Envoyer un test',
		'Enviar una prueba',
		'Test senden'
	] as Row,
	'delivery.sending': [
		'Sending…',
		'Envoi…',
		'Enviando…',
		'Wird gesendet…'
	] as Row,
	'delivery.testDelivered': [
		'Test delivered to {name}.',
		'Test livré à {name}.',
		'Prueba entregada a {name}.',
		'Test an {name} zugestellt.'
	] as Row,
	'delivery.testFailed': [
		'The test delivery failed',
		'La livraison du test a échoué',
		'La entrega de prueba falló',
		'Die Testzustellung ist fehlgeschlagen'
	] as Row,
	'outage.title': [
		'Nothing can be delivered right now',
		'Rien ne peut être livré pour l’instant',
		'Ahora mismo no se puede entregar nada',
		'Im Moment kann nichts zugestellt werden'
	] as Row,
	'outage.text': [
		'Every enabled target is down, so Antenne cannot raise an alert through any channel it has, including the one that would tell you about this. Until one target recovers, this page is the only place a failure will appear.',
		'Toutes les cibles activées sont en panne : Antenne ne peut lever d’alerte par aucun de ses canaux, y compris celui qui vous préviendrait. Tant qu’une cible n’est pas revenue, cette page est le seul endroit où un échec apparaîtra.',
		'Todos los destinos activados están caídos, así que Antenne no puede emitir una alerta por ningún canal, ni siquiera el que te avisaría de esto. Hasta que un destino se recupere, esta página es el único lugar donde aparecerá un fallo.',
		'Alle aktivierten Ziele sind ausgefallen, deshalb kann Antenne über keinen ihrer Kanäle einen Alarm auslösen, auch nicht über den, der dich darüber informieren würde. Bis ein Ziel wieder erreichbar ist, erscheint ein Fehler nur auf dieser Seite.'
	] as Row,
	'outage.notices': [
		{ one: '{count} notice could not be carried anywhere:', other: '{count} notices could not be carried anywhere:' },
		{ one: '{count} notification n’a pu être acheminée nulle part :', other: '{count} notifications n’ont pu être acheminées nulle part :' },
		{ one: '{count} aviso no pudo enviarse a ninguna parte:', other: '{count} avisos no pudieron enviarse a ninguna parte:' },
		{ one: '{count} Benachrichtigung konnte nirgendwohin gesendet werden:', other: '{count} Benachrichtigungen konnten nirgendwohin gesendet werden:' }
	] as Row,
	'outage.since': [
		'{title}, since {when}',
		'{title}, depuis {when}',
		'{title}, desde {when}',
		'{title}, seit {when}'
	] as Row,
	'health.loading': [
		'Loading…',
		'Chargement…',
		'Cargando…',
		'Wird geladen…'
	] as Row,
	'health.ok': [
		'Reachable',
		'Joignable',
		'Accesible',
		'Erreichbar'
	] as Row,
	'health.pending': [
		'Failing, not confirmed yet',
		'En échec, pas encore confirmé',
		'Fallando, aún sin confirmar',
		'Fehlerhaft, noch nicht bestätigt'
	] as Row,
	'health.down': [
		'Down',
		'En panne',
		'Caído',
		'Ausgefallen'
	] as Row,
	'health.downFor': [
		'Down for {duration}',
		'En panne depuis {duration}',
		'Caído desde hace {duration}',
		'Seit {duration} ausgefallen'
	] as Row,
	'health.recovering': [
		'Recovering, answering again',
		'Rétablissement, répond à nouveau',
		'Recuperándose, responde de nuevo',
		'Erholt sich, antwortet wieder'
	] as Row,
	'health.recoveringAfter': [
		'Recovering, answering again after {duration}',
		'Rétablissement, répond à nouveau après {duration}',
		'Recuperándose, responde de nuevo tras {duration}',
		'Erholt sich, antwortet nach {duration} wieder'
	] as Row,
	'health.flapping': [
		'Unstable, breaking and recovering',
		'Instable, tombe et revient',
		'Inestable, se cae y se recupera',
		'Instabil, fällt aus und erholt sich'
	] as Row,
	'health.stale': [
		'Stale, no recent observation',
		'Périmé, aucune observation récente',
		'Obsoleto, sin observación reciente',
		'Veraltet, keine aktuelle Beobachtung'
	] as Row,
	'health.probeError': [
		'Probe failed, Antenne’s side, not the target’s',
		'Sonde en échec, côté Antenne, pas côté cible',
		'La sonda falló, por parte de Antenne, no del destino',
		'Prüfung fehlgeschlagen, Seite von Antenne, nicht des Ziels'
	] as Row,
	'health.unknown': [
		'Never measured',
		'Jamais mesuré',
		'Nunca medido',
		'Nie gemessen'
	] as Row,
	'health.sourceDelivery': [
		'real delivery',
		'livraison réelle',
		'entrega real',
		'echte Zustellung'
	] as Row,
	'health.sourceDeliveryHelp': [
		'Measured by an actual send. The most authoritative signal there is.',
		'Mesuré par un envoi réel. Le signal le plus fiable qui soit.',
		'Medido con un envío real. La señal más fiable que existe.',
		'Durch einen echten Versand gemessen. Das verlässlichste Signal.'
	] as Row,
	'health.sourceProbe': [
		'probe',
		'sonde',
		'sonda',
		'Prüfung'
	] as Row,
	'health.sourceProbeHelp': [
		'Measured by a read-only check, not by delivering anything.',
		'Mesuré par une vérification en lecture seule, sans rien livrer.',
		'Medido con una comprobación de solo lectura, sin entregar nada.',
		'Durch eine reine Leseprüfung gemessen, ohne etwas zuzustellen.'
	] as Row,
	'health.sourceSynthetic': [
		'synthetic send',
		'envoi de test',
		'envío sintético',
		'synthetischer Versand'
	] as Row,
	'health.sourceSyntheticHelp': [
		'Measured by a real message sent to the destination you nominated.',
		'Mesuré par un vrai message envoyé à la destination que vous avez désignée.',
		'Medido con un mensaje real enviado al destino que indicaste.',
		'Durch eine echte Nachricht an das von dir festgelegte Ziel gemessen.'
	] as Row,
	'health.cooldownHelp': [
		'Antenne’s own limit on checks, not a refusal from this target.',
		'Limite d’Antenne sur les vérifications, pas un refus de cette cible.',
		'Límite de Antenne para las comprobaciones, no un rechazo de este destino.',
		'Das Limit von Antenne für Prüfungen, keine Ablehnung dieses Ziels.'
	] as Row,
	'health.checkHelp': [
		'Probe this target right now',
		'Sonder cette cible maintenant',
		'Sondear este destino ahora',
		'Dieses Ziel jetzt prüfen'
	] as Row,
	'health.checking': [
		'Checking…',
		'Vérification…',
		'Comprobando…',
		'Wird geprüft…'
	] as Row,
	'health.retryIn': [
		'Retry in {seconds}s',
		'Réessayer dans {seconds} s',
		'Reintentar en {seconds} s',
		'Erneut in {seconds} s'
	] as Row,
	'health.checkNow': [
		'Check now',
		'Vérifier maintenant',
		'Comprobar ahora',
		'Jetzt prüfen'
	] as Row,
	'health.checked': [
		'Checked {when}',
		'Vérifié {when}',
		'Comprobado {when}',
		'Geprüft {when}'
	] as Row,
	'health.lastWorked': [
		'Last worked {when}',
		'Dernier succès {when}',
		'Último acierto {when}',
		'Zuletzt erfolgreich {when}'
	] as Row,
	'health.noticesSent': [
		'{count} notices sent',
		'{count} notifications envoyées',
		'{count} avisos enviados',
		'{count} Benachrichtigungen gesendet'
	] as Row,
	'health.paused': [
		'Probing paused until {when}. The credential was rejected outright, so retrying it can only get Antenne blocked.',
		'Sondage suspendu jusqu’à {when}. L’identifiant a été rejeté d’emblée, le réessayer ne ferait que faire bloquer Antenne.',
		'Sondeo en pausa hasta {when}. La credencial fue rechazada de plano, así que reintentarla solo haría que bloqueen a Antenne.',
		'Prüfung pausiert bis {when}. Die Zugangsdaten wurden direkt abgelehnt, ein erneuter Versuch würde nur dazu führen, dass Antenne gesperrt wird.'
	] as Row,
	'target.providers': [
		'Providers',
		'Sources',
		'Fuentes',
		'Quellen'
	] as Row,
	'target.system': [
		'Antenne system',
		'Système Antenne',
		'Sistema Antenne',
		'Antenne-System'
	] as Row,
	'target.systemType': [
		'System',
		'Système',
		'Sistema',
		'System'
	] as Row,
	'target.routingHelp': [
		'Routing is opt-in with no fallthrough: a target with nothing selected here and no tag below receives nothing. Antenne’s own events need “Antenne system”.',
		'Le routage est explicite, sans repli : une cible sans rien de coché ici et sans tag ci-dessous ne reçoit rien. Les événements propres à Antenne demandent « Système Antenne ».',
		'El enrutamiento es explícito y sin respaldo: un destino sin nada seleccionado aquí ni etiqueta abajo no recibe nada. Los eventos propios de Antenne requieren «Sistema Antenne».',
		'Das Routing ist ausdrücklich und ohne Auffangregel: Ein Ziel ohne Auswahl hier und ohne Tag unten erhält nichts. Antennes eigene Ereignisse brauchen „Antenne-System“.'
	] as Row,
	'target.providerTags': [
		'Provider tags',
		'Tags de sources',
		'Etiquetas de fuentes',
		'Quellen-Tags'
	] as Row,
	'target.providerTagsHelp': [
		'A provider carrying any of these tags reaches this target, even without being selected above.',
		'Une source portant l’un de ces tags atteint cette cible, même sans être cochée ci-dessus.',
		'Una fuente con cualquiera de estas etiquetas llega a este destino, aunque no esté seleccionada arriba.',
		'Eine Quelle mit einem dieser Tags erreicht dieses Ziel, auch wenn sie oben nicht ausgewählt ist.'
	] as Row,
	'target.homeserver': [
		'Homeserver',
		'Serveur Matrix',
		'Servidor Matrix',
		'Homeserver'
	] as Row,
	'target.homeserverHelp': [
		'Change it only if this target uses another homeserver.',
		'À modifier seulement si cette cible utilise un autre serveur.',
		'Cámbialo solo si este destino usa otro servidor.',
		'Nur ändern, wenn dieses Ziel einen anderen Homeserver nutzt.'
	] as Row,
	'target.accessToken': [
		'Access token',
		'Jeton d’accès',
		'Token de acceso',
		'Zugriffstoken'
	] as Row,
	'target.accessTokenHelp': [
		'Optional when a password login is configured. Used until the homeserver rejects it.',
		'Facultatif si une connexion par mot de passe est configurée. Utilisé jusqu’à ce que le serveur le refuse.',
		'Opcional si hay un inicio de sesión con contraseña. Se usa hasta que el servidor lo rechaza.',
		'Optional, wenn eine Passwort-Anmeldung eingerichtet ist. Wird verwendet, bis der Homeserver es ablehnt.'
	] as Row,
	'target.userId': [
		'User ID',
		'Identifiant utilisateur',
		'ID de usuario',
		'Benutzer-ID'
	] as Row,
	'target.userIdHelp': [
		'With a password, lets Antenne log back in on its own.',
		'Avec un mot de passe, permet à Antenne de se reconnecter seule.',
		'Con una contraseña, permite a Antenne volver a iniciar sesión por sí sola.',
		'Mit einem Passwort kann sich Antenne selbstständig wieder anmelden.'
	] as Row,
	'target.password': [
		'Password',
		'Mot de passe',
		'Contraseña',
		'Passwort'
	] as Row,
	'target.deviceId': [
		'Device ID',
		'Identifiant d’appareil',
		'ID de dispositivo',
		'Geräte-ID'
	] as Row,
	'target.deviceIdHelp': [
		'Optional. Keep it stable so the homeserver reissues for one device.',
		'Facultatif. Gardez-le stable pour que le serveur réémette pour un seul appareil.',
		'Opcional. Mantenlo estable para que el servidor reemita para un solo dispositivo.',
		'Optional. Stabil halten, damit der Homeserver für ein Gerät neu ausstellt.'
	] as Row,
	'target.refreshToken': [
		'Refresh token',
		'Jeton de renouvellement',
		'Token de renovación',
		'Refresh-Token'
	] as Row,
	'target.refreshTokenHelp': [
		'Optional. Used when the access token expires.',
		'Facultatif. Utilisé quand le jeton d’accès expire.',
		'Opcional. Se usa cuando caduca el token de acceso.',
		'Optional. Wird verwendet, wenn das Zugriffstoken abläuft.'
	] as Row,
	'target.roomIds': [
		'Room IDs',
		'Identifiants de salons',
		'IDs de salas',
		'Raum-IDs'
	] as Row,
	'target.roomIdsHelp': [
		'At least one, or this target has nowhere to post.',
		'Au moins un, sinon cette cible n’a nulle part où publier.',
		'Al menos una, o este destino no tiene dónde publicar.',
		'Mindestens eine, sonst kann dieses Ziel nirgends posten.'
	] as Row,
	'target.addRoom': [
		'Add room',
		'Ajouter un salon',
		'Añadir sala',
		'Raum hinzufügen'
	] as Row,
	'target.webhookUrl': [
		'Webhook URL',
		'URL du webhook',
		'URL del webhook',
		'Webhook-URL'
	] as Row,
	'target.webhookUrlHelp': [
		'Required. The URL itself is the credential.',
		'Obligatoire. L’URL elle-même est l’identifiant.',
		'Obligatorio. La propia URL es la credencial.',
		'Erforderlich. Die URL selbst ist das Zugangsdatum.'
	] as Row,
	'target.host': [
		'Host',
		'Hôte',
		'Host',
		'Host'
	] as Row,
	'target.port': [
		'Port',
		'Port',
		'Puerto',
		'Port'
	] as Row,
	'target.implicitTls': [
		'Implicit TLS',
		'TLS implicite',
		'TLS implícito',
		'Implizites TLS'
	] as Row,
	'target.implicitTlsHelp': [
		'Off connects in the clear and upgrades with STARTTLS when offered.',
		'Désactivé, la connexion démarre en clair et passe en STARTTLS si le serveur le propose.',
		'Desactivado, conecta en claro y pasa a STARTTLS si se ofrece.',
		'Ausgeschaltet verbindet im Klartext und wechselt auf STARTTLS, wenn angeboten.'
	] as Row,
	'target.username': [
		'Username',
		'Nom d’utilisateur',
		'Usuario',
		'Benutzername'
	] as Row,
	'target.fromAddress': [
		'From address',
		'Adresse d’expédition',
		'Dirección de origen',
		'Absenderadresse'
	] as Row,
	'target.toAddresses': [
		'To addresses',
		'Adresses de destination',
		'Direcciones de destino',
		'Empfängeradressen'
	] as Row,
	'target.toAddressesHelp': [
		'At least one recipient.',
		'Au moins un destinataire.',
		'Al menos un destinatario.',
		'Mindestens ein Empfänger.'
	] as Row,
	'target.addRecipient': [
		'Add recipient',
		'Ajouter un destinataire',
		'Añadir destinatario',
		'Empfänger hinzufügen'
	] as Row,
	'target.subjectPrefix': [
		'Subject prefix',
		'Préfixe du sujet',
		'Prefijo del asunto',
		'Betreffpräfix'
	] as Row,
	'target.serverUrl': [
		'Server URL',
		'URL du serveur',
		'URL del servidor',
		'Server-URL'
	] as Row,
	'target.topic': [
		'Topic',
		'Sujet',
		'Tema',
		'Thema'
	] as Row,
	'target.priority': [
		'Priority',
		'Priorité',
		'Prioridad',
		'Priorität'
	] as Row,
	'target.priorityHelp': [
		'1 to 5. ntfy’s default is 3.',
		'De 1 à 5. La valeur par défaut de ntfy est 3.',
		'De 1 a 5. El valor por defecto de ntfy es 3.',
		'1 bis 5. Der Standard von ntfy ist 3.'
	] as Row,
	'target.notificationTags': [
		'Notification tags',
		'Tags de notification',
		'Etiquetas de notificación',
		'Benachrichtigungs-Tags'
	] as Row,
	'target.notificationTagsHelp': [
		'ntfy renders these as emoji.',
		'ntfy les affiche sous forme d’emoji.',
		'ntfy las muestra como emoji.',
		'ntfy zeigt diese als Emoji an.'
	] as Row,
	'target.clickUrl': [
		'Click URL',
		'URL au clic',
		'URL al hacer clic',
		'Klick-URL'
	] as Row,
	'target.url': [
		'URL',
		'URL',
		'URL',
		'URL'
	] as Row,
	'target.workspace': [
		'Workspace',
		'Espace de travail',
		'Espacio de trabajo',
		'Arbeitsbereich'
	] as Row,
	'target.signingSecret': [
		'Signing secret',
		'Secret de signature',
		'Secreto de firma',
		'Signaturgeheimnis'
	] as Row,
	'target.signingSecretRequired': [
		'Required. Perception rejects unsigned events.',
		'Obligatoire. Perception rejette les événements non signés.',
		'Obligatorio. Perception rechaza los eventos sin firmar.',
		'Erforderlich. Perception lehnt unsignierte Ereignisse ab.'
	] as Row,
	'target.signingSecretHelp': [
		'Signs timestamp.eventId.body as x-antenne-signature-256. Empty sends unsigned.',
		'Signe timestamp.eventId.body dans x-antenne-signature-256. Vide, l’envoi n’est pas signé.',
		'Firma timestamp.eventId.body como x-antenne-signature-256. Vacío, se envía sin firma.',
		'Signiert timestamp.eventId.body als x-antenne-signature-256. Leer wird unsigniert gesendet.'
	] as Row,
	'target.rawPayload': [
		'Include the raw payload',
		'Inclure la charge brute',
		'Incluir la carga original',
		'Rohdaten einbeziehen'
	] as Row,
	'target.rawPayloadHelp': [
		'Off strips the third-party body and says so in the envelope.',
		'Désactivé, le corps tiers est retiré et l’enveloppe l’indique.',
		'Desactivado, se elimina el cuerpo de terceros y el sobre lo indica.',
		'Ausgeschaltet wird der Fremdinhalt entfernt und im Umschlag vermerkt.'
	] as Row
};
