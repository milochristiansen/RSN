import { useState, useCallback, useRef, useEffect } from "preact/hooks"
import { html, css, Meta, Title } from "/header.js"

import { Fallback } from "/components/Fallback.js"
import { useAuthRedirect } from "/components/AuthRedirectHook.js"

export const Feeds = (props) => {
	useAuthRedirect("/")

	let [data, setData] = useState([])
	let [ok, setOk] = useState(null)
	let interval = useRef(null)

	let update = useCallback(() => {
		fetch("/api/feed/list", {
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

		<section name="feedlist" class=${listCss}>
			${(() => {
				if (ok === true) {
					return data.map(el => html`
						<a href="/read/feed/${el.ID}" key=${el.ID}>
							<span>${el.Name}</span>
							<span>
								${el.ErrorCode != 200 ? html`<span class="error"> (error ${el.ErrorCode})</span>` : ""}
								${el.Paused ? html`<span class="pause"> (paused)</span>` : ""}
							</span>
						</a>
					`)
				} else if (ok !== null) {
					return html`<${Fallback}>Error loading data: ${ok}<//>`
				} else {
					return html`<${Fallback}>Loading feed data...<//>`
				}
			})()}
		</section>
	`
}

export default Feeds

let listCss = css`
	display: flex;
	flex-direction: column;

	a {
		display: flex;
		flex-direction: row;
		justify-content: space-between;

		margin: 2px;
		padding: 5px;

		border-radius: 5px;
		border-style: outset;
		border-color: var(--secondary-color);

		text-decoration: none;

		.pause {
			color: var(--secondary-color);
		}
		.error {
			color: var(--warning-color);
		}
	}
`
