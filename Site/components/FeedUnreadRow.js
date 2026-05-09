import { useState, useCallback } from "preact/hooks"
import { html } from "/header.js"
import { Style } from "/components/Style.js"

import { ReadUnreadButton } from "/components/ReadUnreadButton.js"

const FeedLink = Style.a`
	font-weight: bold;
	color: var(--heading-color);

	text-decoration: none;

	margin-top: 5px;
	margin-bottom: 5px;
`

const ArticleLink = Style.a`
	width: 100%;
	flex: 1;

	text-decoration: none;

	margin: 2px;
	margin-right: 10px;
	padding: 5px;
`

const LoadingDots = Style.strong`
	color: var(--font-color);

	font-size: 32px;
	text-align: center;
`

const FeedName = Style.span`
	display: flex;
	flex-direction: row;
	justify-content: space-around;
`

const ArticleItem = Style.span`
	display: flex;
	flex-direction: row;
	position: relative;

	border-width: 1px;
	border-radius: 7px;
	border-style: groove;
	border-color: var(--heading-color);

	margin-top: 2px;
`

const FeedRow = Style.div`
	display: flex;
	flex-direction: column;

	margin: 2px;
	padding: 5px;

	border-radius: 5px;
	border-style: outset;
	border-color: var(--secondary-color);
`

export const FeedUnreadRow = (props) => {
	let [read, setRead] = useState({})

	let openArticle = useCallback((evnt, id) => {
		setRead(state => ({...state, [id]: true}))
		fetch(`/api/article/read?id=${id}`).then(r => {
			if (!r.ok) {
				setRead(state => ({...state, [id]: false}))
			}
		})
	}, [])

	return html`
		<${FeedRow}>
			<${FeedName}><${FeedLink} href=${`/read/feed/${props.data.FeedID}`}>${props.data.FeedName}</${FeedLink}></${FeedName}>
			${props.data.Articles.map(item => item === null ? html`<${LoadingDots}><//>` : html`
				<${ArticleItem}
					key=${item.ID}
					onread=${() => setRead(state => ({...state, [item.ID]: true}))}
					onunread=${() => setRead(state => ({...state, [item.ID]: false}))}
				>
					<${ArticleLink}
						href=${item.URL}
						rel="noreferrer"
						target="_blank"
						onclick=${(evnt) => openArticle(evnt, item.ID)}
						native
					>${item.Title}</${ArticleLink}>
					<${ReadUnreadButton} state=${read[item.ID] === true} aid=${item.ID}/>
				<//>
			`)}
		<//>
	`
}

export default FeedUnreadRow
