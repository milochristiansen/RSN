import { useState, useCallback, forwardRef, useImperativeHandle } from "preact/hooks"
import { html } from "/header.js"
import { Style } from "/components/Style.js"

const NotifTitle = Style.h2`
	text-align: center;
	font-size: 2.5em;
`

const NotifText = Style.p`
	text-align: center;
	font-size: 1.5em;
`

const NotifInner = Style.div`
	width: 100%;
`

const NotifContainer = Style.div`
	height: 100%;

	border-style: solid;
	border-color: var(--secondary-color);
	border-radius: 25px;
	border-width: 10px;
	background-color: var(--bg-color);
`

export let Notification = forwardRef(function Notification(props, ref) {
	let [data, setData] = useState({})

	useImperativeHandle(ref, () => ({
		Update(d) {
			if (d == null || d == undefined) {
				d = {}
			}
			setData(d)
		}
	}))

	let display = data.Type == undefined ? "none" : "block"

	return html`
		<${NotifContainer} style=${display}>
			<${NotifInner}>
				${(() => {
					if (data.Type == undefined) {
						return ""
					}
					let d = JSON.parse(data.Data)

					new Audio("/stream/assets/ding.mp3").play();
					switch (data.Type) {
					case "sub":
						switch (d.Months) {
						case 0:
							return html`
								<${NotifTitle}>Thank you ${d.Name}</${NotifTitle}>
								<${NotifText}>A shiny new subscriber!</${NotifText}>
							`
						case 1:
							return html`
								<${NotifTitle}>Thank you ${d.Name}</${NotifTitle}>
								<${NotifText}>Subscriber for a whole month!</${NotifText}>
							`
						default:
							return html`
								<${NotifTitle}>Thank you ${d.Name}</${NotifTitle}>
								<${NotifText}>Subscriber for ${d.Months} months!</${NotifText}>
							`
						}
					case "gift":
						return html`
							<${NotifTitle}>Thank you ${d.Name}</${NotifTitle}>
							<${NotifText}>for gifting ${d.Count} subscriptions!</${NotifText}>
						`
					case "bits":
						return html`
							<${NotifTitle}>Thank you ${d.Name}</${NotifTitle}>
							<${NotifText}>for the ${d.Bits} bits!</${NotifText}>
						`
					case "follow":
						return html`
							<${NotifTitle}>Thank you ${d.Name}</${NotifTitle}>
							<${NotifText}>for the follow!</${NotifText}>
						`
					case "raid":
						return html`
							<${NotifTitle}>Thank you ${d.Name}</${NotifTitle}>
							<${NotifText}>for raiding with ${d.Viewers} viewers!</${NotifText}>
						`
					}
				})()}
			<//>
		<//>
	`
})

export default Notification
