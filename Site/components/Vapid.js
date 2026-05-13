
// I would normally just rawdog promises, but I'm using async/await here because I value my sanity.

export const fetchVapidKey = async () => {
	let r = await fetch('/api/push/vapid', {
		credentials: 'include'
	})
	if (!r.ok) {
		throw new Error('Failed to fetch VAPID key')
	}
	let data = await r.json()
	return data.publicKey
}

export const subscribePush = async (uid) => {
	if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
		return
	}

	let vapidKey = await fetchVapidKey()

	let registration = await navigator.serviceWorker.ready
	let existingSubscription = await registration.pushManager.getSubscription()
	if (existingSubscription) {
		await existingSubscription.unsubscribe()
	}
	let subscription = await registration.pushManager.subscribe({
		userVisibleOnly: true,
		applicationServerKey: vapidKey,
	})

	// Using the getKey method gives some insane format and oh god what is it with browser developers and terrible APIs?
	let sub = subscription.toJSON()

	let r = await fetch('/api/push/subscription', {
		method: 'POST',
		credentials: 'include',
		headers: {'Content-Type': 'application/json'},
		body: JSON.stringify({
			endpoint: subscription.endpoint,
			keys: sub.keys.p256dh,
			auth: sub.keys.auth
		})
	})

	localStorage.setItem('pushEndpoint', subscription.endpoint)

	if (!r.ok) {
		throw 'Failed to register push subscription'
	}
	return
}

export const clearBackendSubscription = async (uid) => {
	let r = await fetch('/api/push/subscription', {
		method: 'DELETE',
		credentials: 'include'
	})
	if (!r.ok) {
		throw 'Failed to clear backend push subscription'
	}
	return
}
