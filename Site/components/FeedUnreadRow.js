import { useState, useCallback } from "preact/hooks"
import { html } from "/header.js"

import { ReadUnreadButton } from "/components/ReadUnreadButton.js"

let rowcss = `
	display: flex;
	flex-direction: column;

	margin: 2px;
	padding: 5px;

	border-radius: 5px;
	border-style: outset;
	border-color: var(--secondary-color);

	.feed {
		display: flex;
		flex-direction: row;
		justify-content: space-around;

		a {
			font-weight: bold;
			color: var(--heading-color);

			text-decoration: none;

			margin-top: 5px;
			margin-bottom: 5px;
		}
	}

	strong {
		color: var(--font-color);

		font-size: 32px;
		text-align: center;
	}

	.article {
		display: flex;
		flex-direction: row;
		position: relative;

		border-width: 1px;
		border-radius: 7px;
		border-style: groove;
		border-color: var(--heading-color);

		margin-top: 2px;

		&-link {
			width: 100%;
			flex: 1;

			text-decoration: none;

			margin: 2px;
			margin-right: 10px;
			padding: 5px;
		}
	}
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
		<div style=${rowcss}>
			<span class="feed"><a href=${`/read/feed/${props.data.FeedID}`}>${props.data.FeedName}</a></span>
			${props.data.Articles.map(item => item === null ? html`<strong>\u00B7\u00B7\u00B7</strong>` : html`
				<span
					key=${item.ID}
					class="article"
					onread=${() => setRead(state => ({...state, [item.ID]: true}))}
					onunread=${() => setRead(state => ({...state, [item.ID]: false}))}
				>
					<a
						href=${item.URL}
						rel="noreferrer"
						target="_blank"
						class="article-link"
						onclick=${(evnt) => openArticle(evnt, item.ID)}
						native
					>${item.Title}</a>
					<${ReadUnreadButton} state=${read[item.ID] === true} aid=${item.ID}/>
				</span>
			`)}
		</div>
	`
}

export default FeedUnreadRow
