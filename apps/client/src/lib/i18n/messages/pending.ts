import type { Row } from '../row';

/** The page where the person confirms what an AI client asked Antenne to do. */
export const pending = {
	'pending.title': [
		'Pending actions · Antenne',
		'Actions en attente · Antenne',
		'Acciones pendientes · Antenne',
		'Ausstehende Aktionen · Antenne'
	] as Row,
	'pending.heading': ['Pending actions', 'Actions en attente', 'Acciones pendientes', 'Ausstehende Aktionen'] as Row,
	'pending.intro': [
		'An AI client asked Antenne to do these. Nothing is sent until you confirm.',
		'Un client IA a demandé ces actions à Antenne. Rien n’est envoyé tant que vous ne confirmez pas.',
		'Un cliente de IA pidió estas acciones a Antenne. No se envía nada hasta que confirmes.',
		'Ein KI-Client hat Antenne um diese Aktionen gebeten. Nichts wird gesendet, bevor Sie bestätigen.'
	] as Row,
	'pending.none': ['Nothing is waiting', 'Rien en attente', 'Nada pendiente', 'Nichts ausstehend'] as Row,
	'pending.noneBody': [
		'Actions an AI client asks for appear here until you confirm or dismiss them.',
		'Les actions demandées par un client IA apparaissent ici jusqu’à ce que vous les confirmiez ou les ignoriez.',
		'Las acciones que pide un cliente de IA aparecen aquí hasta que las confirmes o descartes.',
		'Aktionen, um die ein KI-Client bittet, erscheinen hier, bis Sie sie bestätigen oder verwerfen.'
	] as Row,
	'pending.kind.replay': ['Replay event', 'Rejouer l’événement', 'Repetir evento', 'Ereignis erneut senden'] as Row,
	'pending.askedBy': ['Asked by {client}', 'Demandé par {client}', 'Pedido por {client}', 'Angefragt von {client}'] as Row,
	'pending.to': ['To {target}', 'Vers {target}', 'A {target}', 'An {target}'] as Row,
	'pending.replayNote': [
		'Everyone who reads this target will see the event, even if its routing never sent it there.',
		'Toute personne qui lit cette cible verra l’événement, même si le routage ne l’y a jamais envoyé.',
		'Quien lea este destino verá el evento, aunque el enrutamiento nunca lo haya enviado allí.',
		'Alle, die dieses Ziel lesen, sehen das Ereignis, auch wenn das Routing es nie dorthin geschickt hat.'
	] as Row,
	'pending.confirmReplay': ['Send the event', 'Envoyer l’événement', 'Enviar el evento', 'Ereignis senden'] as Row,
	'pending.dismiss': ['Dismiss', 'Ignorer', 'Descartar', 'Verwerfen'] as Row,
	'pending.changed': [
		'This event or target changed after the request. Ask the AI client again.',
		'Cet événement ou cette cible a changé après la demande. Redemandez au client IA.',
		'Este evento o destino cambió después de la solicitud. Vuelve a pedírselo al cliente de IA.',
		'Dieses Ereignis oder Ziel wurde nach der Anfrage geändert. Fragen Sie den KI-Client erneut.'
	] as Row,
	'pending.missing': [
		'This event or target no longer exists.',
		'Cet événement ou cette cible n’existe plus.',
		'Este evento o destino ya no existe.',
		'Dieses Ereignis oder Ziel existiert nicht mehr.'
	] as Row,
	'pending.sent': ['Event sent', 'Événement envoyé', 'Evento enviado', 'Ereignis gesendet'] as Row,
	'pending.dismissed': ['Request dismissed', 'Demande ignorée', 'Solicitud descartada', 'Anfrage verworfen'] as Row,
	'pending.loadFailed': [
		'Could not load the pending actions.',
		'Impossible de charger les actions en attente.',
		'No se pudieron cargar las acciones pendientes.',
		'Die ausstehenden Aktionen konnten nicht geladen werden.'
	] as Row,
	'pending.failed': [
		'That did not work. Try again.',
		'Cela n’a pas fonctionné. Réessayez.',
		'No funcionó. Inténtalo de nuevo.',
		'Das hat nicht geklappt. Versuchen Sie es erneut.'
	] as Row
};
