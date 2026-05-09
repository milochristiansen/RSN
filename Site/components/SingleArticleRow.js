import { useState, useCallback } from "preact/hooks"
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

export const SingleArticleRow = (props) => {
	let [read, setRead] = useState(props.data.Read)

	return html`
		<div
			key=${props.data.ID}
			onread=${() => setRead(true)}
			onunread=${() => setRead(false)}
			style=${rowcss}
		>
			<a
				href=${props.data.URL}
				target="_blank"
				rel="noreferrer"
				class="article"
				native
			>${props.data.Title}</a>
			<${ReadUnreadButton} state=${read} aid=${props.data.ID}/>
		</div>
	`
}

export default SingleArticleRow
