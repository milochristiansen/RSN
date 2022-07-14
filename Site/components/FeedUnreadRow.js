
import { html, css, Component, createRef } from "/header.js"

import ReadUnreadButton from "/components/ReadUnreadButton.js"

const rowcss = css`
	display: flex;
	flex-direction: column;

	margin: 2px;
	padding: 5px;

	border-radius: 5px;
	border-style: outset;
	border-color: var(--secondary-color);

	h4 {
		text-align: center;
		margin-top: 5px;
		margin-bottom: 5px;
	}

	strong {
		color: var(--font-color);

		font-size: 32px;
		text-align: center;
	}

	span {
		display: flex;
		flex-direction: row;
		position: relative;

		border-width: 1px;
		border-radius: 7px;
		border-style: groove;
		border-color: var(--heading-color);

		margin-top: 2px;

		.article {
			width: 100%;
			flex: 1;

			text-decoration: none;

			margin: 2px;
			padding: 5px;
		}
	}
`

function trimPrefix(str, pre) {
	if (str.indexOf(pre) === 0) {
	    str = str.substring(pre.length);
	}
	return str
}

class FeedUnreadRow extends Component {
	constructor() {
		super();
	
		this.root = createRef()

		// We maintain a temporary cache of what articles have been marked read so that there is a visual indicator
		// of which ones are read (and therefore will go away at the next update)
		this.state = {read: {}}
	}

	openArticle(evnt, id) {
		fetch("/api/article/read?id=" + id).then(r => {
			if (r.ok) {
				this.setState(state => ({read: {...state.read, [id]: true}}))
			}
		})
		window.open(this.data.URL, "_blank", "noreferrer");
		evnt.preventDefault()
	}

	render(props, state) {
		let data = props.data
		if (data.length > 5) {
			data = []
			for (let i = 0; i < 3; i++) {
				data[i] = props.data[i]
			}
			data.push(null)
			data.push(props.data[props.data.length - 1])
		}

		return html`
			<div ref=${this.root} class=${rowcss}>
				<h4>${props.data[0].FeedName}</h4>
				${data.map(item => item === null ? html`<strong>\u00B7\u00B7\u00B7</strong>` : html`
					<span
						key=${item.ID}
						onread=${() => this.setState(state => ({read: {...state.read, [item.ID]: true}}))}
						onunread=${() => this.setState(state => ({read: {...state.read, [item.ID]: false}}))}
					>
						<a
							href=${item.URL}
							rel="noreferrer"
							class="article"
							onclick=${(evnt) => this.openArticle(evnt, item.ID)}
							native
						>${trimPrefix(item.Title, props.data[0].FeedName + " - ")}</a>
						<${ReadUnreadButton} state=${this.state.read[item.ID] === true} aid=${item.ID}/>
					</span>
				`)}
			</div>
		`
	}
}

export default FeedUnreadRow