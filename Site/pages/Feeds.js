
import { html, css, Meta, Title } from "/header.js"
import AuthedComponent from "/components/AuthedComponent.js"

import Fallback from "/components/Fallback.js"

class Feeds extends AuthedComponent {
	constructor() {
		super();
	
		this.state = {data: [], ok: null}

		this.update()
	}

	renderAuthed(auth, props, state) {
		return html`
			<${Title} text="RSN - Feeds" />
			<${Meta} k="description" v="Really Simple Notifier subscribed feed list page." />

			<section name="feedlist" class=${this.css.list}>
				${(() => {
					if (state.ok === true) {
						return state.data.map(el => html`
							<a href="/read/feed/${el.ID}" key=${el.ID}>
								<span>${el.Name}</span>
								<span>
									${el.ErrorCode != 200 ? html`<span class="error"> (error ${el.ErrorCode})</span>` : ""}
									${el.Paused ? html`<span class="pause"> (paused)</span>` : ""}
								</span>
							</a>
						`)
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
		fetch("/api/feed/list", {
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
				this.setState({data: data})
			})
	}

	css = {
		list: css`
			display: flex;
			flex-direction: column;

			a {
				display: flex;
				flex-direction: row;
				justify-content: space-between;

				margin: 2px;
				padding: 5px;

				border-radius: 5px;
				border-style: outset;
				border-color: var(--secondary-color);

				text-decoration: none;

				.pause {
					color: var(--secondary-color);
				}
				.error {
					color: var(--warning-color);
				}
			}
		`
	}
}

export default Feeds
