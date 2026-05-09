import { useState, useCallback, useRef, useEffect } from "preact/hooks"
import { html, Meta, Title, Style } from "/header.js"

import { FeedRecentReadRow } from "/components/FeedRecentReadRow.js"
import { Fallback } from "/components/Fallback.js"
import { useAuthRedirect } from "/components/AuthRedirectHook.js"

export const RecentRead = (props) => {
	useAuthRedirect("/")

	let [data, setData] = useState([])
	let [ok, setOk] = useState(null)
	let interval = useRef(null)

	let update = useCallback(() => {
		fetch("/api/recentread", {
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
		<${Title} text="RSN - Recently Read" />
		<${Meta} k="description" v="Really Simple Notifier recently read articles page." />

		<${RecentReadList}>
			${(() => {
				if (ok === true) {
					return data.map(el => html`<${FeedRecentReadRow} data=${el} key=${el.ID}/>`)
				} else if (ok !== null) {
					return html`<${Fallback}>Error loading data: ${ok}<//>`
				} else {
					return html`<${Fallback}>Loading article data...<//>`
				}
			})()}
		<//>
	`
}

export default RecentRead

let RecentReadList = Style.section`
	display: flex;
	flex-direction: column;
`
