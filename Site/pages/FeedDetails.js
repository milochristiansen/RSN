
import { html, css, Meta, Title } from "/header.js"
import AuthedComponent from "/components/AuthedComponent.js"

import SingleArticleRow from "/components/SingleArticleRow.js"

class FeedDetails extends AuthedComponent {
	constructor(props) {
		super();
	
		this.state = {data: {}, articles: []}

		this.update(props.id)
	}

	renderAuthed(auth, props, state) {
		return html`
			<${Title} text="RSN - Feed Details" />
			<${Meta} k="description" v="Really Simple Notifier feed details page." />

			<section name="feed-details" class=${this.css.details}>
				<h2 class="row">${state.data.Name} ${state.data.Paused && html`<span>(paused)</span>`}</h2>
				<a class="row" href=${state.data.URL}>${state.data.URL}</a>
				${state.data.ErrorCode != 200 ? html`<span class="row">Feed currently down, code ${state.data.ErrorCode}</span>` : ""}
				${this.isrr() && html`<a href=${this.isrr()}>Go to Fiction Page on Royal Road</a>`}
			</section>
			<section name="feed-article-list" class=${this.css.list}>
				${state.articles.map(el => html`<${SingleArticleRow} key=${el.ID} data=${el} />`)}
			</section>
		`;
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

			h2, a {
				width: 100%;
				overflow: wrap;
				overflow-wrap: break-word;

				text-decoration: none;

				padding-left: 10px;
				padding-right: 10px;
			
				margin-bottom: 10px;
			}
		`,
		list: css`
			display: flex;
			flex-direction: column;
		`
	}
}

export default FeedDetails
