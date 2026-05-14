self.addEventListener('push', (event) => {
    console.log("Event: ", event)
    const data = event.data ? event.data.json() : { ts: Date.now(), msg: 'New notification' }
    console.log("Data:", data)
    event.waitUntil(
        console.log("Sent.")
        self.registration.showNotification('RSN', {
            body: data.msg,
            icon: '/assets/round.svg',
            badge: '/assets/favicon.svg',
            timestamp: data.ts,
            requireInteraction: true,
        })
    )
})
