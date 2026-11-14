import type { Row } from '../row';

/** The signed-out landing page and its demos. */
export const landing = {
	'landing.title': [
		'Antenne · Every alert in the right place',
		'Antenne · Chaque alerte au bon endroit',
		'Antenne · Cada alerta en su sitio',
		'Antenne · Jede Warnung am richtigen Ort'
	] as Row,
	'landing.nav.features': ['Features', 'Fonctionnalités', 'Funciones', 'Funktionen'] as Row,
	'landing.nav.downloads': ['Downloads', 'Téléchargements', 'Descargas', 'Downloads'] as Row,
	'downloads.title': ['Download Antenne', 'Télécharger Antenne', 'Descargar Antenne', 'Antenne herunterladen'] as Row,
	'downloads.line': [
		'The CLI follows the log, tests targets and manages API keys from your terminal.',
		'Le CLI suit le journal, teste les cibles et gère les clés d\'API depuis votre terminal.',
		'La CLI sigue el registro, prueba destinos y gestiona claves de API desde tu terminal.',
		'Die CLI liest das Protokoll mit, testet Ziele und verwaltet API-Schlüssel im Terminal.'
	] as Row,
	'landing.hero.title': [
		'Every alert in the right place',
		'Chaque alerte au bon endroit',
		'Cada alerta en su sitio',
		'Jede Warnung am richtigen Ort'
	] as Row,
	'landing.hero.line': [
		'Antenne watches your services and sends each alert to Matrix, Discord, ntfy or email.',
		'Antenne surveille vos services et envoie chaque alerte vers Matrix, Discord, ntfy ou par e-mail.',
		'Antenne vigila tus servicios y envía cada alerta a Matrix, Discord, ntfy o por correo.',
		'Antenne überwacht deine Dienste und schickt jede Warnung an Matrix, Discord, ntfy oder per E-Mail.'
	] as Row,
	'landing.hero.cli': ['Install the CLI', 'Installer le CLI', 'Instalar la CLI', 'CLI installieren'] as Row,
	'landing.features.title': ['What Antenne does', 'Ce que fait Antenne', 'Lo que hace Antenne', 'Was Antenne macht'] as Row,
	'landing.feature.sources.title': [
		'One log for every source',
		'Un journal pour toutes les sources',
		'Un registro para todas las fuentes',
		'Ein Protokoll für alle Quellen'
	] as Row,
	'landing.feature.sources.body': [
		'Websites, mailboxes, RSS feeds and webhooks land in the same activity log.',
		'Sites web, boîtes mail, flux RSS et webhooks arrivent dans le même journal.',
		'Sitios web, buzones, feeds RSS y webhooks llegan al mismo registro.',
		'Websites, Postfächer, RSS-Feeds und Webhooks landen im selben Protokoll.'
	] as Row,
	'landing.feature.routing.title': [
		'Tags decide who hears what',
		'Les tags décident qui reçoit quoi',
		'Las etiquetas deciden quién recibe qué',
		'Tags entscheiden, wer was erfährt'
	] as Row,
	'landing.feature.routing.body': [
		'A target gets an event only when it names the source or shares a tag.',
		'Une cible reçoit un événement seulement si elle nomme la source ou partage un tag.',
		'Un destino recibe un evento solo si nombra la fuente o comparte una etiqueta.',
		'Ein Ziel erhält ein Ereignis nur, wenn es die Quelle nennt oder einen Tag teilt.'
	] as Row,
	'landing.feature.retry.title': ['Nothing gets lost', 'Rien ne se perd', 'Nada se pierde', 'Nichts geht verloren'] as Row,
	'landing.feature.retry.body': [
		'Antenne retries failed deliveries, and you can replay any event.',
		'Antenne relance les envois ratés, et vous pouvez rejouer chaque événement.',
		'Antenne reintenta los envíos fallidos y puedes reenviar cualquier evento.',
		'Antenne wiederholt fehlgeschlagene Zustellungen, und jedes Ereignis lässt sich erneut senden.'
	] as Row,
	'landing.feature.bus.title': [
		'The event bus of the suite',
		'Le bus d\'événements de la suite',
		'El bus de eventos de la suite',
		'Der Event-Bus der Suite'
	] as Row,
	'landing.feature.bus.body': [
		'Sablier, Sonde, Agenda and the other Facile apps talk through one pool.',
		'Sablier, Sonde, Agenda et les autres apps Facile échangent sur un même pool.',
		'Sablier, Sonde, Agenda y las demás apps de Facile se comunican por un mismo pool.',
		'Sablier, Sonde, Agenda und die anderen Facile-Apps sprechen über einen Pool.'
	] as Row,
	'landing.cli.title': [
		'Follow it from the terminal',
		'Suivez tout depuis le terminal',
		'Síguelo desde la terminal',
		'Alles im Terminal mitlesen'
	] as Row,
	'landing.cli.tail': [
		'Streams the activity log live, one line per event.',
		'Affiche le journal en direct, une ligne par événement.',
		'Muestra el registro en directo, una línea por evento.',
		'Zeigt das Protokoll live, eine Zeile pro Ereignis.'
	] as Row,
	'landing.cta.title': [
		'Send your first alert',
		'Envoyez votre première alerte',
		'Envía tu primera alerta',
		'Sende deine erste Warnung'
	] as Row,
	'landing.demo.label': [
		'Two alerts arrive in the Antenne activity log, and a failed delivery is replayed.',
		'Deux alertes arrivent dans le journal d\'Antenne, et un envoi raté est rejoué.',
		'Dos alertas llegan al registro de Antenne y se reenvía una entrega fallida.',
		'Zwei Warnungen kommen im Antenne-Protokoll an, und eine fehlgeschlagene Zustellung wird erneut gesendet.'
	] as Row,
	'landing.demo.down': [
		'api.facile.studio is down',
		'api.facile.studio ne répond plus',
		'api.facile.studio no responde',
		'api.facile.studio ist nicht erreichbar'
	] as Row,
	'landing.demo.deploy': [
		'Deploy finished',
		'Déploiement terminé',
		'Despliegue terminado',
		'Deployment abgeschlossen'
	] as Row
};
