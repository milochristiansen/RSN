import { useState, useCallback, useEffect } from "preact/hooks"
import { html, Meta, Title, Style } from "/header.js"
import { useLocation } from 'preact-iso';
import { SingleArticleRow } from "/components/SingleArticleRow.js"
import { Fallback } from "/components/Fallback.js"
import { useAuthRedirect } from "/components/AuthRedirectHook.js"

let DetailRow = Style.div`
	width: 100%;
	overflow: wrap;
	overflow-wrap: break-word;

	text-decoration: none;

	padding-left: 10px;
	padding-right: 10px;

	margin-bottom: 10px;
`

let DetailError = Style.span`
	color: var(--warning-color);
`

let ActionButton = Style.button`
	padding: 5px;
	padding-left: 30px;
	padding-right: 30px;

	margin-left: 10px;
	margin-right: 10px;
`

let DeleteButton = Style.button`
	padding: 5px;
	padding-left: 30px;
	padding-right: 30px;

	margin-left: 10px;
	margin-right: 10px;

	&.confirm {
		border-color: var(--warning-color);
	}
`

let ButtonGroup = Style.span`
	display: flex;
	flex-direction: row;
	justify-content: center;

	margin-top: 10px;
`

let FeedArticleList = Style.section`
	display: flex;
	flex-direction: column;
`

let FeedDetailsSection = Style.section`
	display: flex;
	flex-direction: column;
	text-align: center;
`

let RenameForm = Style.form`
	display: flex;
	flex-direction: row;
	justify-content: center;

	width: 80%;

	margin-left: auto;
	margin-right: auto;

	margin-top: 20px;
`

let RenameInput = Style.input`
	padding: 5px;
	margin-right: 10px;
	flex: 1;
`

let RenameStatus = Style.span`
	margin-top: 5px;
	margin-bottom: 15px;

	&.error {
		color: var(--warning-color);
	}
`

export const FeedDetails = (props) => {
	useAuthRedirect("/")
	let { route } = useLocation()

	let [data, setData] = useState({})
	let [articles, setArticles] = useState([])
	let [deleteConfirm, setDeleteConfirm] = useState(false)
	let [dataOk, setDataOk] = useState(null)
	let [artOk, setArtOk] = useState(null)
	let [renameValue, setRenameValue] = useState("")
	let [renameMsg, setRenameMsg] = useState({ text: "", type: "" })

	let update = useCallback((id, all) => {
		fetch("/api/feed/details?id="+id, {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					setDataOk(s => {
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
				setDataOk(true)
			})

		if (!all) {
			return
		}

		fetch("/api/feed/articles?id="+id, {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					setArtOk(s => {
						if (s === null) {
							return r.status
						}
						return s
					})
					throw new Error(r.status)
				}
				return r.json()
			})
			.then(articles => {
				setArticles(articles)
				setArtOk(true)
			})
	}, [])

	let pause = useCallback((shouldPause) => {
		let url = `/api/feed/unpause?id=${props.id}`
		if (shouldPause) {
			url = `/api/feed/pause?id=${props.id}`
		}
		fetch(url).then(r => {
			if (r.ok) {
				update(props.id, false)
			}
		})
	}, [props.id, update])

	let deleteFeed = useCallback(() => {
		if (!deleteConfirm) {
			setDeleteConfirm(true)
			setTimeout(() => setDeleteConfirm(false), 2000)
			return
		}

		let url = `/api/feed/unsubscribe?id=${props.id}`
		fetch(url).then(r => {
			if (r.ok) {
				route("/read/feeds")
			}
		})
		setDeleteConfirm(false)
	}, [props.id, deleteConfirm])

	let renameFeed = useCallback((e) => {
		e.preventDefault()
		if (!renameValue.trim()) {
			setRenameMsg({ text: "Name cannot be empty", type: "error" })
			setTimeout(() => setRenameMsg({ text: "", type: "" }), 3000)
			return
		}

		fetch(`/api/feed/rename?id=${props.id}&name=${encodeURIComponent(renameValue)}`, {
			method: 'POST',
			credentials: 'include'
		}).then(r => {
			if (r.ok) {
				update(props.id, false)
				setRenameValue("")
				setRenameMsg({ text: "Feed renamed successfully", type: "success" })
				setTimeout(() => setRenameMsg({ text: "", type: "" }), 3000)
			} else {
				setRenameMsg({ text: "Failed to rename feed", type: "error" })
				setTimeout(() => setRenameMsg({ text: "", type: "" }), 3000)
			}
		}).catch(() => {
			setRenameMsg({ text: "Network error", type: "error" })
			setTimeout(() => setRenameMsg({ text: "", type: "" }), 3000)
		})
	}, [props.id, renameValue, update])

	let isrr = useCallback(() => {
		if (!data.URL) {
			return null
		}

		let info = data.URL.match(/https:\/\/www\.royalroad\.com\/fiction\/syndication\/([0-9]+)/)
		if (info === null) {
			return null
		}
		return `https://www.royalroad.com/fiction/${info[1]}`
	}, [data.URL])

	useEffect(() => {
		update(props.id, true)
	}, [props.id, update])

	return html`
		<${Title} text="RSN - Feed Details" />
		<${Meta} k="description" v="Really Simple Notifier feed details page." />

		<${FeedDetailsSection}>
			${(() => {
				if (dataOk === true) {
					return html`
						<${DetailRow}><h2>${data.Name} ${data.Paused && html`<span>(paused)</span>`}</h2></${DetailRow}>
						<${DetailRow}><a target="_blank" rel="noreferrer" href=${data.URL} native>${data.URL}</a></${DetailRow}>
						${data.ErrorCode != 200 ? html`<${DetailError}>Feed currently down, code ${data.ErrorCode}</${DetailError}>` : ""}
						${isrr() && html`<${DetailRow}><a target="_blank" rel="noreferrer" href=${isrr()} native>Go to Fiction Page on Royal Road</a></${DetailRow}>`}
						<${ButtonGroup}>
							${data.Paused ?
								html`<${ActionButton} onclick=${() => pause(false)}>Unpause Feed</${ActionButton}>` :
								html`<${ActionButton} onclick=${() => pause(true)}>Pause Feed</${ActionButton}>`
							}
							<${DeleteButton} onclick=${deleteFeed} class=${deleteConfirm ? "confirm" : ""}>Delete Feed</${DeleteButton}>
						<//>
						<${RenameForm} onsubmit=${renameFeed}>
							<${RenameInput} type="text" placeholder="New feed name" value=${renameValue} oninput=${(e) => setRenameValue(e.target.value)} />
							<${ActionButton} type="submit">Rename Feed</${ActionButton}>
						<//>
						<${RenameStatus} class=${renameMsg.type === "error" ? "error" : ""}>${renameMsg.text != "" ? renameMsg.text : ""}${"\u200b"}<//>
					`
				} else if (dataOk !== null) {
					return html`<${Fallback}>Error loading data: ${dataOk}<//>`
				} else {
					return html`<${Fallback}>Loading feed data...<//>`
				}
			})()}
		<//>
		<${FeedArticleList}>
			${(() => {
				if (artOk === true) {
					return articles.map(el => html`<${SingleArticleRow} key=${el.ID} data=${el} />`)
				} else if (artOk !== null) {
					return html`<${Fallback}>Error loading data: ${artOk}<//>`
				} else {
					return html`<${Fallback}>Loading article data...<//>`
				}
			})()}
		<//>
	`
}

export default FeedDetails
