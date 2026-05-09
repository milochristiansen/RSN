import { useState, useCallback, forwardRef, useImperativeHandle } from "preact/hooks"
import { html } from "/header.js"

let bodyCss = `
	height: 100%;

	border-style: solid;
	border-color: var(--secondary-color);
	border-radius: 25px;
	border-width: 10px;
	background-color: var(--bg-color);

	.inner {
		width: 100%;
	}

	h2, p {
		text-align: center;
	}
	h2 {
		font-size: 2.5em;
	}
	p {
		font-size: 1.5em;
	}
`

let Notification = forwardRef(function Notification(props, ref) {
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
		<div style="${bodyCss}; display: ${display}">
			<div class="inner">
				${() => {
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
								<h2>Thank you ${d.Name}</h2>
								<p>A shiny new subscriber!</p>
							`
						case 1:
							return html`
								<h2>Thank you ${d.Name}</h2>
								<p>Subscriber for a whole month!</p>
							`
						default:
							return html`
								<h2>Thank you ${d.Name}</h2>
								<p>Subscriber for ${d.Months} months!</p>
							`
						}
					case "gift":
						return html`
							<h2>Thank you ${d.Name}</h2>
							<p>for gifting ${d.Count} subscriptions!</p>
						`
					case "bits":
						return html`
							<h2>Thank you ${d.Name}</h2>
							<p>for the ${d.Bits} bits!</p>
						`
					case "follow":
						return html`
							<h2>Thank you ${d.Name}</h2>
							<p>for the follow!</p>
						`
					case "raid":
						return html`
							<h2>Thank you ${d.Name}</h2>
							<p>for raiding with ${d.Viewers} viewers!</p>
						`
					}
				}()}
			</div>
		</div>
	`
})

export default Notification
