
import { html, css, Meta, Title } from "/header.js"
import AuthedComponent from "/components/AuthedComponent.js"

import FeedRecentReadRow from "/components/FeedRecentReadRow.js"

class RecentRead extends AuthedComponent {
	constructor() {
		super();
	
		this.interval = null

		this.state = {data: []}

		this.update()
	}

	renderAuthed(auth, props, state) {
		return html`
			<${Title} text="RSN - Recently Read" />
			<${Meta} k="description" v="Really Simple Notifier recently read articles page." />

			<section name="unreadlist" class=${this.css.list}>
				${state.data.map(el => html`<${FeedRecentReadRow} data=${el} key=${el.ID}/>`)}
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
					throw new Error("Request failed.")
				}
				return r.json()
			})
			.then(data => {
				this.setState({data: data})
			})
			.catch(err => {
				console.log(err)
			})
	}

	componentDidMount() {
		this.interval = setInterval(this.update, 60000)
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
