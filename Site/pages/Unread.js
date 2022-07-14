
import { html, css, Meta, Title } from "/header.js"
import AuthedComponent from "/components/AuthedComponent.js"

import FeedUnreadRow from "/components/FeedUnreadRow.js"

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
						return state.data.map(el => html`<${FeedUnreadRow} data=${el} key=${el[0].FeedID} />`)
					} else if (state.ok === false) {
						return html`<span>Error loading data: ${state.ok}</span>`
					} else {
						return html`<span>Loading feed data...</span>`
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
				// Fill the state with the freshly downloaded data.
				// Since we get the data sorted by date without being split by feed, we can just do splitting by feed
				// here and not need to do any sorting. The feeds come out sorted by first article, and the articles
				// inside the feed come out sorted by date.
				let newdata = []
				let helper = {}
				for (const el of data) {
					const fi = helper[el.FeedID]
					if (fi != undefined) {
						newdata[fi].push(el)
						continue
					}
					helper[el.FeedID] = newdata.length
					newdata.push([el])
				}
				this.setState({data: newdata, ok: true})
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
