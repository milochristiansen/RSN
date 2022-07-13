
import { html, css, Component } from "/header.js"

class AddFeed extends Component {
	constructor() {
		super();
	
		this.state = {addstate: null, url: "", name: ""}
	}

	addfeed(evnt) {
		evnt.preventDefault()

		if (this.state.url == "" || this.state.name == "") {
			this.setState({addstate: false})
			setTimeout(() => (this.setState({addstate: null})), 5000);
			return;
		}

		let self = this;
		fetch("/api/feed/subscribe", {
			method: "POST",
			credentials: "include",
			body: JSON.stringify({
				URL: String(this.url),
				Name: String(this.name)
			})
		})
			.then(function(res) {
				if (res.ok) {
					this.setState({addstate: true, url: "", name: ""})
					setTimeout(() => (this.setState({addstate: null})), 3000);
					return;
				}
				throw new Error(res.status);
			})
			.catch(error => {
				console.error(error.message);
				this.setState({addstate: false})
				setTimeout(() => (this.setState({addstate: null})), 5000);
			});
	}

	handleInput(e) {
		this.setState({[e.target.name]: e.target.value})
	}

	render(props, state) {
		let status = html`<span> </span>`
		if (state.addstate === false) {
			status = html`<span class="error">Failed adding feed.</span>`
		} else if (state.addstate === true) {
			status = html`<span>Feed added!</span>`
		}

		return html`
			<section name="addfeed" class=${this.css.body}>
				<div class="status">
					${status}
				</div>
				<form onsubmit=${e => this.addfeed(e)} class=${this.css.form}>
					<input type="text" placeholder="Feed URL" name="url" value=${state.url} onInput=${e => this.handleInput(e)} />
					<input type="text" placeholder="Feed Name" name="name" value=${state.name} onInput=${e => this.handleInput(e)} />
					<input type="submit" value="Subscribe Feed" />
				</form>
			</section>
		`
	}

	css = {
		body: css`
			margin-top: 5px;
			margin-bottom: 10px;

			margin-left: 10px;
			margin-right: 10px;
		`,
		form: css`
			display: flex;
			flex-direction: column;

			input {
				margin-top: 2px;
			}
		`
	}
}

export default AddFeed
