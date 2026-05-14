import { useState, useCallback, useRef, useEffect } from "preact/hooks"
import { html, Meta, Title, Style } from "/header.js"

import { Fallback } from "/components/Fallback.js"
import { useAuthRedirect } from "/components/AuthRedirectHook.js"

let FeedLabel = Style.span`

`

let StatusLabel = Style.span`
	&.pause {
		color: var(--secondary-color);
	}
	&.error {
		color: var(--warning-color);
	}
`

let FeedLink = Style.a`
	display: flex;
	flex-direction: row;
	justify-content: space-between;

	margin: 2px;
	padding: 5px;

	border-radius: 5px;
	border-style: outset;
	border-color: var(--secondary-color);

	text-decoration: none;
`

let FeedList = Style.section`
	display: flex;
	flex-direction: column;
`

export const Feeds = (props) => {
	useAuthRedirect("/")

	let [data, setData] = useState([])
	let [ok, setOk] = useState(null)
	let interval = useRef(null)

	let update = useCallback(() => {
		fetch("/api/feeds", {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					setOk(s => {
						if (s === null) {
							return r.status
						}
						return s
					})
					throw new Error(r.status)
				}
				return r.json()
			})
			.then(data => {
				setData(data)
				setOk(true)
			})
	}, [])

	useEffect(() => {
		update()
	}, [update])

	return html`
		<${Title} text="RSN - Feeds" />
		<${Meta} k="description" v="Really Simple Notifier subscribed feed list page." />

		<${FeedList}>
			${(() => {
				if (ok === true) {
					return data.map(el => html`
						<${FeedLink} href="/read/feed/${el.ID}" key=${el.ID}>
							<${FeedLabel}>${el.Name}</${FeedLabel}>
							<${FeedLabel}>
								${el.ErrorCode != 200 ? (
									el.ErrorCode < 1000 ? (
										html`<${StatusLabel} class="error"> (error ${el.ErrorCode})</${StatusLabel}>`
									) : (
										html`<${StatusLabel} class="error"> (non-HTTP error)</${StatusLabel}>`
									)
								) : ""}
								${el.Paused ? html`<${StatusLabel} class="pause"> (paused)</${StatusLabel}>` : ""}
							</${FeedLabel}>
						</${FeedLink}>
					`)
				} else if (ok !== null) {
					return html`<${Fallback}>Error loading data: ${ok}<//>`
				} else {
					return html`<${Fallback}>Loading feed data...<//>`
				}
			})()}
		<//>
	`
}

export default Feeds
