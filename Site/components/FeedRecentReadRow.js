
import { html, css, Component, createRef } from "/header.js"

import ReadUnreadButton from "/components/ReadUnreadButton.js"

const rowcss = css`
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

class FeedRecentReadRow extends Component {
	constructor() {
		super();
	
		this.root = createRef()

		// We maintain a temporary cache of what articles have been marked read so that there is a visual indicator
		// of which ones are read (and therefore will go away at the next update)
		this.state = {read: true}
	}

	render(props, state) {
		let item = props.data

		return html`
			<div
				onread=${() => this.setState(state => ({read: true}))}
				onunread=${() => this.setState(state => ({read: false}))}
				class=${rowcss}
			>
				<a
					href=${item.URL}
					target="_blank"
					rel="noreferrer"
					class="article"
					native
				>${item.FeedName} - ${item.Title}</a>
				<${ReadUnreadButton} state=${this.state.read} aid=${item.ID}/>
			</div>
		`
	}
}

export default FeedRecentReadRow
