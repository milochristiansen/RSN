self.addEventListener('push', (event) => {
    const data = event.data ? event.data.json() : { ts: Date.now(), msg: 'New notification' }
    event.waitUntil(
        self.registration.showNotification('RSN', {
            body: data.msg,
            icon: '/assets/round.svg',
            badge: '/assets/favicon.svg',
            requireInteraction: true,
        })
    )
})

// DO NOT USE THIS WITH CACHING! Push events only.
self.addEventListener('activate', async () => {
    self.skipWaiting()
})
