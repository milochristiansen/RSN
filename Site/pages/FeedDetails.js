
import { html, css, Meta, Title } from "/header.js"
import { route } from 'preact-router';
import AuthedComponent from "/components/AuthedComponent.js"

import SingleArticleRow from "/components/SingleArticleRow.js"

class FeedDetails extends AuthedComponent {
	constructor(props) {
		super();
	
		this.state = {data: {}, articles: [], delete: false, dataOk: null, artOk: null}

		this.update(props.id)
	}

	renderAuthed(auth, props, state) {
		return html`
			<${Title} text="RSN - Feed Details" />
			<${Meta} k="description" v="Really Simple Notifier feed details page." />

			<section name="feed-details" class=${this.css.details}>
				${(() => {
					if (state.dataOk === true) {
						return html`
							<h2 class="row">${state.data.Name} ${state.data.Paused && html`<span>(paused)</span>`}</h2>
							<a class="row" href=${state.data.URL}>${state.data.URL}</a>
							${state.data.ErrorCode != 200 ? html`<span class="row error">Feed currently down, code ${state.data.ErrorCode}</span>` : ""}
							${this.isrr() && html`<a class="row" href=${this.isrr()}>Go to Fiction Page on Royal Road</a>`}
							<span class="row buttons">
								${state.data.Paused ?
									html`<button onclick=${() => this.pause(true)}>Unpause Feed</button>` :
									html`<button onclick=${() => this.pause(false)}>Pause Feed</button>`
								}
								<button onclick=${() => this.delete()} class=${state.delete ? "confirm" : ""}>Delete Feed</button>
							</span>
						`
					} else if (state.dataOk !== null) {
						return html`<span class="status">Error loading data: ${state.dataOk}</span>`
					} else {
						return html`<span class="status">Loading feed data...</span>`
					}
				})()}
			</section>
			<section name="feed-article-list" class=${this.css.list}>
				${(() => {
					if (state.artOk === true) {
						return state.articles.map(el => html`<${SingleArticleRow} key=${el.ID} data=${el} />`)
					} else if (state.artOk !== null) {
						return html`<span class="status">Error loading data: ${state.artOk}</span>`
					} else {
						return html`<span class="status">Loading article data...</span>`
					}
				})()}
			</section>
		`;
	}

	pause(y) {
		let url = `/api/feed/unpause?id=${this.props.id}`
		if (y) {
			url = `/api/feed/pause?id=${this.props.id}`
		}
		fetch(url).then(r => {
			if (r.ok) {
				this.update(props.id)
			}
		})
	}

	delete() {
		this.setState(state => {
			if (!this.state.delete) {
				setTimeout(() => this.setState({delete: false}), 2000)
				return {delete: true}
			}

			let url = `/api/feed/unsubscribe?id=${this.props.id}`
			fetch(url).then(r => {
				if (r.ok) {
					route("/read/feeds")
				}
			})
			return {delete: false}
		})
	}

	isrr() {
		if (!this.state.data.URL) {
			return null
		}

		let info = this.state.data.URL.match(/https:\/\/www\.royalroad\.com\/fiction\/syndication\/([0-9]+)/)
		if (info === null) {
			return null
		}
		return `https://www.royalroad.com/fiction/${info[1]}`
	}

	noAuthRedirect() {
		return "/"
	}

	update(id) {
		fetch("/api/feed/details?id="+id, {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					this.setState(state => {
						if (state.dataOk === null) {
							return {dataOk: r.status}
						}
						return {} // Change nothing
					})
					throw new Error(r.status)
				}
				return r.json()
			})
			.then(data => {
				this.setState({data: data, dataOk: true})
			})

		fetch("/api/feed/articles?id="+id, {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					this.setState(state => {
						if (state.artOk === null) {
							return {artOk: r.status}
						}
						return {} // Change nothing
					})
					throw new Error(r.status)
				}
				return r.json()
			})
			.then(articles => {
				this.setState({articles: articles, artOk: true})
			})
	}

	css = {
		details: css`
			display: flex;
			flex-direction: column;
			text-align: center;

			.row {
				width: 100%;
				overflow: wrap;
				overflow-wrap: break-word;

				text-decoration: none;

				padding-left: 10px;
				padding-right: 10px;
			
				margin-bottom: 10px;
			}

			.error {
				color: var(--warning-color);
			}

			.buttons {
				display: flex;
				flex-direction: row;
				justify-content: center;

				button {
					padding: 5px;
					padding-left: 30px;
					padding-right: 30px;

					margin-left: 10px;
					margin-right: 10px;
				}
			}

			.confirm {
				border-color: var(--warning-color);
			}

			.status {
				width: 100%;
				font-size: 32px;
				text-align: center;
			}
		`,
		list: css`
			display: flex;
			flex-direction: column;

			.status {
				width: 100%;
				font-size: 32px;
				text-align: center;
			}
		`
	}
}

export default FeedDetails
