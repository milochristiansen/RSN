import { useState, useCallback } from "preact/hooks"
import { html } from "/header.js"
import { Style } from "/components/Style.js"

import { ReadUnreadButton } from "/components/ReadUnreadButton.js"

const ArticleLink = Style.a`
	width: 100%;
	flex: 1;

	text-decoration: none;

	margin: 2px;
	padding: 5px;
`

const ArticleRow = Style.div`
	display: flex;
	flex-direction: row;

	margin: 2px;
	padding: 5px;

	border-radius: 5px;
	border-style: outset;
	border-color: var(--secondary-color);
`

export const SingleArticleRow = (props) => {
	let [read, setRead] = useState(props.data.Read)

	return html`
		<${ArticleRow}
			key=${props.data.ID}
			onread=${() => setRead(true)}
			onunread=${() => setRead(false)}
		>
			<${ArticleLink}
				href=${props.data.URL}
				target="_blank"
				rel="noreferrer"
				native
			>${props.data.Title}</${ArticleLink}>
			<${ReadUnreadButton} state=${read} aid=${props.data.ID}/>
		<//>
	`
}

export default SingleArticleRow
