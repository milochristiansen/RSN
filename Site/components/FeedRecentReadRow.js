import { useState } from "preact/hooks"
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

const RecentRow = Style.div`
	display: flex;
	flex-direction: row;

	margin: 2px;
	padding: 5px;

	border-radius: 5px;
	border-style: outset;
	border-color: var(--secondary-color);
`

export const FeedRecentReadRow = (props) => {
	let [read, setRead] = useState(true)

	let item = props.data

	return html`
		<${RecentRow}
			onread=${() => setRead(true)}
			onunread=${() => setRead(false)}
		>
			<${ArticleLink}
				href=${item.URL}
				target="_blank"
				rel="noreferrer"
				native
			>${item.FeedName} - ${item.Title}</${ArticleLink}>
			<${ReadUnreadButton} state=${read} aid=${item.ID}/>
		<//>
	`
}

export default FeedRecentReadRow
