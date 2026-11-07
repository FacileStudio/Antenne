import type { Row } from '../row';

/** Activity log, charts and the target detail page. */
export const activity = {
	'activity.title': [
		'Activity',
		'Activité',
		'Actividad',
		'Aktivität'
	] as Row,
	'activity.subtitle': [
		'Every alert Antenne received, and how each delivery went.',
		'Chaque alerte reçue par Antenne, et le déroulé de chaque livraison.',
		'Cada alerta que recibió Antenne y cómo fue cada entrega.',
		'Jeder Alarm, den Antenne erhalten hat, und wie jede Zustellung verlief.'
	] as Row,
	'activity.noEventsYet': [
		'No events yet',
		'Aucun événement',
		'Aún no hay eventos',
		'Noch keine Ereignisse'
	] as Row,
	'activity.refresh': [
		'Refresh',
		'Actualiser',
		'Actualizar',
		'Aktualisieren'
	] as Row,
	'activity.refreshing': [
		'Refreshing…',
		'Actualisation…',
		'Actualizando…',
		'Wird aktualisiert…'
	] as Row,
	'activity.tracked': [
		'Tracked',
		'Suivis',
		'Registrados',
		'Erfasst'
	] as Row,
	'activity.hitRate': [
		'Hit rate',
		'Taux de réussite',
		'Tasa de éxito',
		'Erfolgsquote'
	] as Row,
	'activity.recent': [
		'Recent (6h)',
		'Récents (6 h)',
		'Recientes (6 h)',
		'Zuletzt (6 Std.)'
	] as Row,
	'activity.stream': [
		'Activity stream',
		'Flux d’activité',
		'Flujo de actividad',
		'Aktivitätsstrom'
	] as Row,
	'activity.live': [
		'Live: new events arrive without a refresh.',
		'En direct : les nouveaux événements arrivent sans actualisation.',
		'En vivo: los eventos nuevos llegan sin actualizar.',
		'Live: neue Ereignisse erscheinen ohne Aktualisierung.'
	] as Row,
	'activity.search': [
		'Search events…',
		'Rechercher des événements…',
		'Buscar eventos…',
		'Ereignisse suchen…'
	] as Row,
	'activity.loadMore': [
		'Load more',
		'Charger plus',
		'Cargar más',
		'Mehr laden'
	] as Row,
	'activity.loadFailed': [
		'Failed to load the activity log',
		'Impossible de charger le journal d’activité',
		'No se pudo cargar el registro de actividad',
		'Das Aktivitätsprotokoll konnte nicht geladen werden'
	] as Row,
	'activity.noTarget': [
		'No delivery target configured.',
		'Aucune cible de livraison configurée.',
		'No hay ningún destino de entrega configurado.',
		'Kein Zustellziel eingerichtet.'
	] as Row,
	'activity.replayed': [
		'Replayed to {name}.',
		'Rejoué vers {name}.',
		'Reenviado a {name}.',
		'An {name} erneut gesendet.'
	] as Row,
	'activity.replayFailed': [
		'The replay failed',
		'La relecture a échoué',
		'El reenvío falló',
		'Das erneute Senden ist fehlgeschlagen'
	] as Row,
	'activity.allSources': [
		'All sources',
		'Toutes les sources',
		'Todas las fuentes',
		'Alle Quellen'
	] as Row,
	'source.webhook': [
		'Webhook',
		'Webhook',
		'Webhook',
		'Webhook'
	] as Row,
	'source.website': [
		'Website',
		'Site web',
		'Sitio web',
		'Website'
	] as Row,
	'source.imap': [
		'Inbox',
		'Boîte mail',
		'Buzón',
		'Postfach'
	] as Row,
	'source.rss': [
		'RSS',
		'RSS',
		'RSS',
		'RSS'
	] as Row,
	'source.system': [
		'System',
		'Système',
		'Sistema',
		'System'
	] as Row,
	'source.test': [
		'Test',
		'Test',
		'Prueba',
		'Test'
	] as Row,
	'events.emptyTitle': [
		'No events recorded yet',
		'Aucun événement enregistré',
		'Aún no hay eventos registrados',
		'Noch keine Ereignisse aufgezeichnet'
	] as Row,
	'events.emptyText': [
		'Every alert Antenne receives, and every delivery it attempts, lands here.',
		'Chaque alerte reçue par Antenne et chaque livraison tentée arrive ici.',
		'Cada alerta que recibe Antenne y cada entrega que intenta aparece aquí.',
		'Jeder Alarm, den Antenne erhält, und jede versuchte Zustellung landet hier.'
	] as Row,
	'events.replayHelp': [
		'Send this event again',
		'Renvoyer cet événement',
		'Enviar este evento de nuevo',
		'Dieses Ereignis erneut senden'
	] as Row,
	'events.replay': [
		'Replay',
		'Rejouer',
		'Reenviar',
		'Erneut senden'
	] as Row,
	'chart.last24': [
		'Last 24 hours',
		'Dernières 24 heures',
		'Últimas 24 horas',
		'Letzte 24 Stunden'
	] as Row,
	'chart.delivered': [
		'Delivered',
		'Livrés',
		'Entregados',
		'Zugestellt'
	] as Row,
	'chart.withErrors': [
		'With errors',
		'Avec erreurs',
		'Con errores',
		'Mit Fehlern'
	] as Row,
	'chart.recent': [
		{ one: '{count} event in 6h', other: '{count} events in 6h' },
		{ one: '{count} événement en 6 h', other: '{count} événements en 6 h' },
		{ one: '{count} evento en 6 h', other: '{count} eventos en 6 h' },
		{ one: '{count} Ereignis in 6 Std.', other: '{count} Ereignisse in 6 Std.' }
	] as Row,
	'chart.peak': [
		'Peak {count} at {hour}',
		'Pic de {count} à {hour}',
		'Pico de {count} a las {hour}',
		'Spitze {count} um {hour}'
	] as Row,
	'chart.empty': [
		'Nothing in the last 24 hours',
		'Rien au cours des dernières 24 heures',
		'Nada en las últimas 24 horas',
		'Nichts in den letzten 24 Stunden'
	] as Row,
	'targets.back': [
		'Back to delivery',
		'Retour à la livraison',
		'Volver a la entrega',
		'Zurück zur Zustellung'
	] as Row,
	'targets.editTarget': [
		'Edit target',
		'Modifier la cible',
		'Editar destino',
		'Ziel bearbeiten'
	] as Row,
	'targets.history': [
		'Everything Antenne has tried to deliver here.',
		'Tout ce qu’Antenne a tenté de livrer ici.',
		'Todo lo que Antenne ha intentado entregar aquí.',
		'Alles, was Antenne hier zuzustellen versucht hat.'
	] as Row,
	'targets.goneTitle': [
		'This delivery target no longer exists',
		'Cette cible de livraison n’existe plus',
		'Este destino de entrega ya no existe',
		'Dieses Zustellziel gibt es nicht mehr'
	] as Row,
	'targets.goneText': [
		'It was deleted, or the link points at an id this Antenne never had. The log below is empty for the same reason.',
		'Elle a été supprimée, ou le lien pointe vers un identifiant que cette instance n’a jamais eu. Le journal ci-dessous est vide pour la même raison.',
		'Se eliminó, o el enlace apunta a un id que esta instancia nunca tuvo. El registro de abajo está vacío por el mismo motivo.',
		'Es wurde gelöscht, oder der Link verweist auf eine ID, die diese Instanz nie hatte. Das Protokoll unten ist aus demselben Grund leer.'
	] as Row,
	'targets.log': [
		'Delivery log',
		'Journal de livraison',
		'Registro de entrega',
		'Zustellprotokoll'
	] as Row,
	'targets.logText': [
		'The last 100 events routed to this target.',
		'Les 100 derniers événements routés vers cette cible.',
		'Los últimos 100 eventos enrutados a este destino.',
		'Die letzten 100 Ereignisse, die an dieses Ziel geroutet wurden.'
	] as Row,
	'targets.emptyTitle': [
		'Nothing delivered yet',
		'Rien de livré pour l’instant',
		'Aún no se ha entregado nada',
		'Noch nichts zugestellt'
	] as Row,
	'targets.emptyText': [
		'Events routed to this target show up here, successful or not.',
		'Les événements routés vers cette cible apparaissent ici, réussis ou non.',
		'Los eventos enrutados a este destino aparecen aquí, con éxito o no.',
		'Ereignisse, die an dieses Ziel geroutet werden, erscheinen hier, ob erfolgreich oder nicht.'
	] as Row,
	'targets.loadFailed': [
		'Failed to load the delivery log',
		'Impossible de charger le journal de livraison',
		'No se pudo cargar el registro de entrega',
		'Das Zustellprotokoll konnte nicht geladen werden'
	] as Row,
	'targets.deleteText': [
		'{name} stops receiving alerts immediately.',
		'{name} cesse immédiatement de recevoir des alertes.',
		'{name} deja de recibir alertas de inmediato.',
		'{name} erhält sofort keine Alarme mehr.'
	] as Row
};
