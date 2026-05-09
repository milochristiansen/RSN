import { useState, useCallback, useEffect } from "preact/hooks"
import { html, css, Meta, Title } from "/header.js"
import { route } from 'preact-router';
import { SingleArticleRow } from "/components/SingleArticleRow.js"
import { Fallback } from "/components/Fallback.js"
import { useAuthRedirect } from "/components/AuthRedirectHook.js"

export const FeedDetails = (props) => {
	useAuthRedirect("/")

	let [data, setData] = useState({})
	let [articles, setArticles] = useState([])
	let [deleteConfirm, setDeleteConfirm] = useState(false)
	let [dataOk, setDataOk] = useState(null)
	let [artOk, setArtOk] = useState(null)

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

		<section name="feed-details" class=${detailsCss}>
			${(() => {
				if (dataOk === true) {
					return html`
						<h2 class="row">${data.Name} ${data.Paused && html`<span>(paused)</span>`}</h2>
						<a class="row" target="_blank" rel="noreferrer" href=${data.URL} native>${data.URL}</a>
						${data.ErrorCode != 200 ? html`<span class="row error">Feed currently down, code ${data.ErrorCode}</span>` : ""}
						${isrr() && html`<a class="row" target="_blank" rel="noreferrer" href=${isrr()} native>Go to Fiction Page on Royal Road</a>`}
						<span class="row buttons">
							${data.Paused ?
								html`<button onclick=${() => pause(false)}>Unpause Feed</button>` :
								html`<button onclick=${() => pause(true)}>Pause Feed</button>`
							}
							<button onclick=${deleteFeed} class=${deleteConfirm ? "confirm" : ""}>Delete Feed</button>
						</span>
						<!--<span class="row buttons">
							<input type="input" name="rename"></input>
							<button onclick=${() => {}} class="">Rename Feed</button>
						</span>-->
					`
				} else if (dataOk !== null) {
					return html`<${Fallback}>Error loading data: ${dataOk}<//>`
				} else {
					return html`<${Fallback}>Loading feed data...<//>`
				}
			})()}
		</section>
		<section name="feed-article-list" class=${listCss}>
			${(() => {
				if (artOk === true) {
					return articles.map(el => html`<${SingleArticleRow} key=${el.ID} data=${el} />`)
				} else if (artOk !== null) {
					return html`<${Fallback}>Error loading data: ${artOk}<//>`
				} else {
					return html`<${Fallback}>Loading article data...<//>`
				}
			})()}
		</section>
	`
}

export default FeedDetails

let detailsCss = css`
	display: flex;
	flex-direction: column;
	text-align: center;

	.row {
		width: 100%;
		overflow: wrap;
		overflow-wrap: break-word;

		text-decoration: none;

		padding-left: 10px;
		padding-right: 10px;
	
		margin-bottom: 10px;
	}

	.error {
		color: var(--warning-color);
	}

	.buttons {
		display: flex;
		flex-direction: row;
		justify-content: center;

		margin-top: 10px;
		margin-bottom: 15px;

		button {
			padding: 5px;
			padding-left: 30px;
			padding-right: 30px;

			margin-left: 10px;
			margin-right: 10px;
		}
	}

	.confirm {
		border-color: var(--warning-color);
	}
`

let listCss = css`
	display: flex;
	flex-direction: column;
`
