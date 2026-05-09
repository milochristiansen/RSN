import { useState } from "preact/hooks"
import { html } from "/header.js"

import { ReadUnreadButton } from "/components/ReadUnreadButton.js"

let rowcss = `
	display: flex;
	flex-direction: row;

	margin: 2px;
	padding: 5px;

	border-radius: 5px;
	border-style: outset;
	border-color: var(--secondary-color);

	.article {
		width: 100%;
		flex: 1;

		text-decoration: none;

		margin: 2px;
		padding: 5px;
	}
`

export const FeedRecentReadRow = (props) => {
	let [read, setRead] = useState(true)

	let item = props.data

	return html`
		<div
			onread=${() => setRead(true)}
			onunread=${() => setRead(false)}
			style=${rowcss}
		>
			<a
				href=${item.URL}
				target="_blank"
				rel="noreferrer"
				class="article"
				native
			>${item.FeedName} - ${item.Title}</a>
			<${ReadUnreadButton} state=${read} aid=${item.ID}/>
		</div>
	`
}

export default FeedRecentReadRow
