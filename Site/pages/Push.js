import { useState, useCallback, useEffect } from "preact/hooks"
import { html, Meta, Title, Style } from "/header.js"

import { useAuthRedirect } from "/components/AuthRedirectHook.js"
import { AuthConsumer } from "/components/Auth.js"
import { subscribePush, clearBackendSubscription } from "/components/Vapid.js"

let Body = Style.section`
	max-width: 600px;
`

let Error = Style.p`
	.error {
		color: var(--warning-color);
	}
`

export const Push = (props) => {
	useAuthRedirect("/")

	let [msg, setMsg] = useState({ text: "", type: "" })

	const handleSubscribe = (e, auth) => {
		e.preventDefault()
		setMsg({ text: "Subscribing...", type: "" })
		subscribePush()
			.then(() => {
				auth.refresh()
				setMsg({ text: "Subscribed successfully", type: "" })
				setTimeout(() => setMsg({ text: "", type: "" }), 3000)
			})
			.catch(() => {
				setMsg({ text: "Failed to subscribe", type: "error" })
				setTimeout(() => setMsg({ text: "", type: "" }), 3000)
			})
	}

	const handleDisable = (e, auth) => {
		e.preventDefault()
		setMsg({ text: "Disabling...", type: "" })
		clearBackendSubscription()
			.then(() => {
				localStorage.removeItem('pushEndpoint')
				auth.refresh()
				setMsg({ text: "Push notifications disabled", type: "" })
				setTimeout(() => setMsg({ text: "", type: "" }), 3000)
			})
			.catch(() => {
				setMsg({ text: "Failed to disable", type: "error" })
				setTimeout(() => setMsg({ text: "", type: "" }), 3000)
			})
	}

	return html`
		<${Title} text="RSN - Push Notifications" />
		<${Meta} k="description" v="Manage push notifications settings." />

		<${AuthConsumer}>
			${auth => {
				if (!auth.ok) {
					return html`<${Fallback}>Loading...<//>`
				}
				
				const whoami = auth.whoami
				const subscribed = whoami?.pushSubscribed || false
				const pushSupported = 'PushManager' in window && 'serviceWorker' in navigator
				const permission = Notification.permission
				
				return html`
					<${Body}>
						<h2>Push Notifications</h2>
						${subscribed ? html`
							<p>You are receiving push notifications on this device.</p>
							<button onclick=${(e) => handleDisable(e, auth)}>Disable Push Notifications</button>
						` : html`
							<p>Enable push notifications to receive updates when new articles are posted.</p>
							${!pushSupported && html`<p>Push notifications are not supported in your browser. Please use a modern browser that supports the Push API.</p>`}
							${permission === "not-granted" && html`<p>To allow notifications, click the lock icon in the address bar and change notification settings to Allow.</p>`}
							${permission === "denied" && html`<p>You previously blocked notifications for this site. To allow notifications, go to your browser settings and find this site in the notification permissions list. Change the setting to Allow.</p>`}
							<button onclick=${(e) => handleSubscribe(e, auth)}>Enable Push Notifications</button>
						`}
						${msg.text !== "" && html`<${Error} class=${msg.type === "error" ? "error" : ""}>${msg.text}${"\u200b"}<//>`}
					<//>
				`
			}}
		<//>
	`
}

export default Push
