
import { html, css, Meta, Title } from "/header.js"
import AuthedComponent from "/components/AuthedComponent.js"

import FeedUnreadRow from "/components/FeedUnreadRow.js"
import Fallback from "/components/Fallback.js"

class Unread extends AuthedComponent {
	constructor() {
		super();
	
		this.interval = null

		this.state = {data: [], ok: null}

		this.update(true)
	}

	renderAuthed(auth, props, state) {
		return html`
			<${Title} text="RSN - Unread" />
			<${Meta} k="description" v="Really Simple Notifier unread articles page." />

			<section name="unreadlist" class=${this.css.list}>
				${(() => {
					if (state.ok === true) {
						return state.data.map(el => html`<${FeedUnreadRow} data=${el} key=${el.FeedID} />`)
					} else if (state.ok !== null) {
						return html`<${Fallback}>Error loading data: ${state.ok}<//>`
					} else {
						return html`<${Fallback}>Loading feed data...<//>`
					}
				})()}
			</section>
		`;
	}

	noAuthRedirect() {
		return "/"
	}

	update() {
		fetch("/api/getunread", {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					this.setState(state => {
						if (state.ok === null) {
							return {ok: r.status}
						}
						return {} // Change nothing
					})
					throw new Error(r.status)
				}
				return r.json()
			})
			.then(data => {
				this.setState({data: data, ok: true})
			})
	}

	componentDidMount() {
		this.interval = setInterval(() => this.update(), 60000)
	}

	componentWillUnmount() {
		clearInterval(this.interval)
	}

	css = {
		list: css`
			display: flex;
			flex-direction: column;
		`
	}
}

export default Unread
