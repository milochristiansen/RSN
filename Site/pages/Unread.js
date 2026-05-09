import { useState, useCallback, useRef, useEffect } from "preact/hooks"
import { html, Meta, Title, Style } from "/header.js"

import { FeedUnreadRow } from "/components/FeedUnreadRow.js"
import { Fallback } from "/components/Fallback.js"
import { useAuthRedirect } from "/components/AuthRedirectHook.js"

export const Unread = (props) => {
	useAuthRedirect("/")

	let [data, setData] = useState([])
	let [ok, setOk] = useState(null)
	let interval = useRef(null)

	let update = useCallback(() => {
		fetch("/api/getunread", {
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

	useEffect(() => {
		interval.current = setInterval(() => update(), 60000)
		return () => clearInterval(interval.current)
	}, [update])

	return html`
		<${Title} text="RSN - Unread" />
		<${Meta} k="description" v="Really Simple Notifier unread articles page." />

		<${UnreadList}>
			${(() => {
				if (ok === true) {
					return data.map(el => html`<${FeedUnreadRow} data=${el} key=${el.FeedID} />`)
				} else if (ok !== null) {
					return html`<${Fallback}>Error loading data: ${ok}<//>`
				} else {
					return html`<${Fallback}>Loading feed data...<//>`
				}
			})()}
		<//>
	`
}

export default Unread

let UnreadList = Style.section`
	display: flex;
	flex-direction: column;
`
