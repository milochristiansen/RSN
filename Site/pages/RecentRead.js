
import { html, css, Meta, Title } from "/header.js"
import AuthedComponent from "/components/AuthedComponent.js"

import FeedRecentReadRow from "/components/FeedRecentReadRow.js"
import Fallback from "/components/Fallback.js"

class RecentRead extends AuthedComponent {
	constructor() {
		super();
	
		this.interval = null

		this.state = {data: [], ok: null}

		this.update()
	}

	renderAuthed(auth, props, state) {
		return html`
			<${Title} text="RSN - Recently Read" />
			<${Meta} k="description" v="Really Simple Notifier recently read articles page." />

			<section name="unreadlist" class=${this.css.list}>
				${(() => {
					if (state.ok === true) {
						return state.data.map(el => html`<${FeedRecentReadRow} data=${el} key=${el.ID}/>`)
					} else if (state.ok !== null) {
						return html`<${Fallback}>Error loading data: ${state.ok}<//>`
					} else {
						return html`<${Fallback}>Loading article data...<//>`
					}
				})()}
			</section>
		`;
	}

	noAuthRedirect() {
		return "/"
	}

	update() {
		fetch("/api/recentread", {
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

export default RecentRead
