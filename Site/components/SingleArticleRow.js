
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

class SingleArticleRow extends Component {
	constructor(props) {
		super();
	
		this.root = createRef()

		this.state = {read: props.data.Read}
	}

	render(props, state) {
		return html`
			<div
				key=${props.data.ID}
				onread=${() => this.setState(state => ({read: false}))}
				onunread=${() => this.setState(state => ({read: true}))}
				class=${rowcss}
			>
				<a
					href=${props.data.URL}
					target="_blank"
					rel="noreferrer"
					class="article"
					native
				>${props.data.Title}</a>
				<${ReadUnreadButton} state=${this.state.read} aid=${props.data.ID}/>
			</div>
		`
	}
}

export default SingleArticleRow
