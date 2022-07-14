
import { html, css, Meta, Title } from "/header.js"
import { route } from 'preact-router';
import AuthedComponent from "/components/AuthedComponent.js"

import SingleArticleRow from "/components/SingleArticleRow.js"

class FeedDetails extends AuthedComponent {
	constructor(props) {
		super();
	
		this.state = {data: {}, articles: [], delete: false}

		this.update(props.id)
	}

	renderAuthed(auth, props, state) {
		return html`
			<${Title} text="RSN - Feed Details" />
			<${Meta} k="description" v="Really Simple Notifier feed details page." />

			<section name="feed-details" class=${this.css.details}>
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
			</section>
			<section name="feed-article-list" class=${this.css.list}>
				${state.articles.map(el => html`<${SingleArticleRow} key=${el.ID} data=${el} />`)}
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

		fetch("/api/feed/articles?id="+id, {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					throw new Error("Request failed.")
				}
				return r.json()
			})
			.then(articles => {
				this.setState({articles: articles})
			})
			.catch(err => {
				console.log(err)
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
		`,
		list: css`
			display: flex;
			flex-direction: column;
		`
	}
}

export default FeedDetails
